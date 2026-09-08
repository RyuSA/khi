// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package khitools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	inspectioncore_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/impl"
)

// parameterItem describes one inspection parameter to the calling model.
//
// It flattens the nested form structure: groups become a Parent reference rather than nesting,
// because a model reading nested JSON reliably misses the ids buried in the children arrays.
type parameterItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Type is the KHI form field type: text, set, file or group.
	Type string `json:"type"`
	// ValueJSONType is the JSON type this parameter's value must have in the values object.
	// A group carries no value and reports an empty string.
	ValueJSONType string `json:"valueJSONType,omitempty"`
	// ValueFormat names an additional convention the value must follow, when there is one.
	ValueFormat string `json:"valueFormat,omitempty"`
	Description string `json:"description,omitempty"`
	// Parent is the id of the group this parameter belongs to, when it is nested.
	Parent string `json:"parent,omitempty"`
	// Default is the value KHI uses when the parameter is absent from the values object.
	Default any `json:"default,omitempty"`
	// Options are the selectable values of a set parameter.
	Options []string `json:"options,omitempty"`
	// AllowCustomValue reports whether a set parameter accepts values outside Options.
	AllowCustomValue bool `json:"allowCustomValue,omitempty"`
	// Suggestions are autocompletion candidates of a text parameter, resolved from live state.
	Suggestions []string `json:"suggestions,omitempty"`
	// Readonly parameters are fixed by the server configuration and must not be set.
	Readonly bool `json:"readonly,omitempty"`
	// HintType is one of none, info, warning or error. An error blocks the command generation.
	HintType string `json:"hintType,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// parameterProblem points at a parameter whose current value is not acceptable.
type parameterProblem struct {
	ParameterID string `json:"parameterId"`
	Message     string `json:"message"`
}

// queryItem is a log query KHI would issue with the current parameters.
type queryItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Query          string `json:"query"`
	EstimatedCount *int64 `json:"estimatedCount,omitempty"`
	Incomplete     bool   `json:"incomplete,omitempty"`
}

// prepareJobCommandResult is the khi_prepare_job_command result.
type prepareJobCommandResult struct {
	InspectionType  string   `json:"inspectionType"`
	EnabledFeatures []string `json:"enabledFeatures"`
	// Ready reports whether every parameter is acceptable and JobCommand is therefore filled in.
	Ready          bool               `json:"ready"`
	Parameters     []parameterItem    `json:"parameters"`
	BlockingErrors []parameterProblem `json:"blockingErrors"`
	Warnings       []parameterProblem `json:"warnings,omitempty"`
	Queries        []queryItem        `json:"queries,omitempty"`
	// JobCommand is the runnable command line. It is empty until Ready is true.
	JobCommand        string `json:"jobCommand,omitempty"`
	ExportDestination string `json:"exportDestination,omitempty"`
	NextStep          string `json:"nextStep"`
}

// prepareJobCommandArguments are the khi_prepare_job_command arguments.
type prepareJobCommandArguments struct {
	InspectionType    string         `json:"inspectionType"`
	Features          []string       `json:"features"`
	Values            map[string]any `json:"values"`
	ExportDestination string         `json:"exportDestination"`
}

// prepareJobCommandTool resolves the parameter schema, validates the given values and generates
// the job mode command line. All three come out of a single dry run, so splitting them into
// separate tools would make the caller pay for the same cloud round trips several times over.
type prepareJobCommandTool struct {
	deps Dependencies
}

func (t *prepareJobCommandTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:  "khi_prepare_job_command",
		Title: "Discover KHI parameters and build the job mode command",
		Description: "Returns the parameters an inspection needs, validates the values given so far, and " +
			"once nothing is left to fix returns the runnable \"khi --job-mode\" command line. Call it with " +
			"an empty values object to discover the parameters, then repeatedly with more values until " +
			"\"ready\" is true; \"blockingErrors\" names exactly what is still missing. This is the " +
			"authoritative parameter schema, which depends on the inspection type, the enabled features and " +
			"live cloud state, so never guess parameter ids. Requires Google Cloud credentials for the " +
			"\"gcp-*\" inspection types.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "inspectionType": {
      "type": "string",
      "description": "An id returned by khi_list_inspection_types, for example \"gcp-gke\"."
    },
    "features": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Feature ids returned by khi_list_features, or the single element \"ALL\" to enable every feature. Omit to use the features enabled by default."
    },
    "values": {
      "type": "object",
      "additionalProperties": true,
      "description": "Map of parameter id to value. Start with {} to discover the parameters. Each value must match the parameter's valueJSONType exactly: \"string\" takes a JSON string, \"string[]\" takes a JSON array of strings.",
      "default": {}
    },
    "exportDestination": {
      "type": "string",
      "description": "Path the generated command writes the .khi file to. Defaults to \"output.khi\"."
    }
  },
  "required": ["inspectionType"],
  "additionalProperties": false
}`),
		// The dry run reaches live cloud APIs for autocompletion and log volume estimation, but it
		// writes nothing: KHI gates every persisting write on the run mode.
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: true},
	}
}

func (t *prepareJobCommandTool) Call(ctx context.Context, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	args := prepareJobCommandArguments{}
	if err := decodeArguments(arguments, &args); err != nil {
		return mcp.ErrorResult("%v", err), nil
	}
	if args.InspectionType == "" {
		return mcp.ErrorResult("inspectionType is required. %s", t.knownInspectionTypesHint()), nil
	}
	if args.Values == nil {
		args.Values = map[string]any{}
	}
	exportDestination := args.ExportDestination
	if exportDestination == "" {
		exportDestination = t.deps.DefaultExportDestination
	}

	inspectionID, err := t.deps.InspectionServer.CreateInspection(args.InspectionType)
	if err != nil {
		return mcp.ErrorResult("unknown inspection type %q. %s", args.InspectionType, t.knownInspectionTypesHint()), nil
	}
	defer t.deps.InspectionServer.DeleteInspection(inspectionID)
	runner := t.deps.InspectionServer.GetInspection(inspectionID)

	features, err := runner.ResolveFeatureList(args.Features)
	if err != nil {
		return mcp.ErrorResult("failed to resolve the feature list: %v", err), nil
	}
	if err := runner.SetFeatureList(features); err != nil {
		return mcp.ErrorResult("failed to enable the requested features: %v\nCall khi_list_features for %q to get the valid feature ids, and copy them verbatim.", err, args.InspectionType), nil
	}

	dryRunResult, err := runner.DryRun(ctx, &inspectioncore_contract.InspectionRequest{Values: args.Values})
	if err != nil {
		return mcp.ErrorResult("%s", describeDryRunError(err)), nil
	}
	metadata, isMap := dryRunResult.Metadata.(map[string]any)
	if !isMap {
		return mcp.ErrorResult("the dry run returned an unexpected metadata shape %T", dryRunResult.Metadata), nil
	}

	parameters := flattenFormFields(readFormFields(metadata), "")
	blockingErrors, warnings := collectParameterProblems(parameters)
	ready := len(blockingErrors) == 0

	result := prepareJobCommandResult{
		InspectionType:  args.InspectionType,
		EnabledFeatures: features,
		Ready:           ready,
		Parameters:      parameters,
		BlockingErrors:  blockingErrors,
		Warnings:        warnings,
		Queries:         readQueries(metadata),
	}
	if ready {
		// The command is generated here rather than taken from the dry run metadata, because the
		// generator used for the web UI replaces every file field with a placeholder path. The web
		// UI only holds an upload token, but the caller of this tool already gave a real local path
		// and overwriting it would produce a command that cannot run.
		command, err := inspectioncore_impl.GenerateJobModeCommand(inspectioncore_impl.JobModeCommandOptions{
			BinaryPath:        t.deps.BinaryPath,
			InspectionType:    args.InspectionType,
			EnabledFeatures:   features,
			Values:            args.Values,
			ExportDestination: exportDestination,
		})
		if err != nil {
			return mcp.ErrorResult("failed to generate the job mode command: %v", err), nil
		}
		result.JobCommand = command
		result.ExportDestination = exportDestination
		result.NextStep = fmt.Sprintf("Run jobCommand with your shell. It writes %s, which you can then open in the KHI web UI.", exportDestination)
	} else {
		result.NextStep = "Set a value for every parameter listed in blockingErrors, then call khi_prepare_job_command again with the same inspectionType and features."
	}
	return mcp.JSONResult(result)
}

// knownInspectionTypesHint lists the valid inspection type ids, so a wrong id is self correcting.
func (t *prepareJobCommandTool) knownInspectionTypesHint() string {
	ids := []string{}
	for _, inspectionType := range t.deps.InspectionServer.GetAllInspectionTypes() {
		ids = append(ids, inspectionType.Id)
	}
	return fmt.Sprintf("Valid inspection types are: %s.", strings.Join(ids, ", "))
}

// readFormFields extracts the form field list from the dry run metadata.
func readFormFields(metadata map[string]any) []inspectionmetadata.ParameterFormField {
	fields, isFields := metadata[inspectionmetadata.FormFieldSetMetadataKey.Key()].([]inspectionmetadata.ParameterFormField)
	if !isFields {
		return nil
	}
	return fields
}

// readQueries extracts the generated log queries from the dry run metadata.
func readQueries(metadata map[string]any) []queryItem {
	items, isItems := metadata[inspectionmetadata.QueryMetadataKey.Key()].([]*inspectionmetadata.QueryItem)
	if !isItems {
		return nil
	}
	queries := make([]queryItem, 0, len(items))
	for _, item := range items {
		queries = append(queries, queryItem{
			ID:             item.Id,
			Name:           item.Name,
			Query:          item.Query,
			EstimatedCount: item.EstimatedCount,
			Incomplete:     item.Incomplete,
		})
	}
	return queries
}

// flattenFormFields turns the nested form field tree into a flat list carrying parent references.
func flattenFormFields(fields []inspectionmetadata.ParameterFormField, parent string) []parameterItem {
	items := []parameterItem{}
	for _, field := range fields {
		base := inspectionmetadata.GetParameterFormFieldBase(field)
		item := parameterItem{
			ID:          base.ID,
			Label:       base.Label,
			Type:        string(base.Type),
			Description: base.Description,
			Parent:      parent,
			HintType:    string(base.HintType),
			Hint:        base.Hint,
		}
		switch typed := field.(type) {
		case inspectionmetadata.TextParameterFormField:
			item.ValueJSONType = "string"
			item.Readonly = typed.Readonly
			if typed.Default != "" {
				item.Default = typed.Default
			}
			item.Suggestions = typed.Suggestions
		case inspectionmetadata.SetParameterFormField:
			item.ValueJSONType = "string[]"
			item.AllowCustomValue = typed.AllowCustomValue
			if len(typed.Default) > 0 {
				item.Default = typed.Default
			}
			for _, option := range typed.Options {
				item.Options = append(item.Options, option.ID)
			}
		case inspectionmetadata.FileParameterFormField:
			item.ValueJSONType = "string"
			item.ValueFormat = "localFilePath"
			if item.Description == "" {
				item.Description = "A path to the file on the machine that will run the generated command."
			}
		case inspectionmetadata.GroupParameterFormField:
			items = append(items, item)
			items = append(items, flattenFormFields(typed.Children, base.ID)...)
			continue
		}
		items = append(items, item)
	}
	return items
}

// collectParameterProblems splits the parameter hints into blocking errors and warnings.
// KHI reports validation results as per field hints rather than as errors, so this is where a
// dry run turns into a yes or no answer about whether the command can be generated.
func collectParameterProblems(parameters []parameterItem) ([]parameterProblem, []parameterProblem) {
	blockingErrors := []parameterProblem{}
	warnings := []parameterProblem{}
	for _, parameter := range parameters {
		if parameter.Hint == "" {
			continue
		}
		switch inspectionmetadata.ParameterHintType(parameter.HintType) {
		case inspectionmetadata.Error:
			blockingErrors = append(blockingErrors, parameterProblem{ParameterID: parameter.ID, Message: parameter.Hint})
		case inspectionmetadata.Warning:
			warnings = append(warnings, parameterProblem{ParameterID: parameter.ID, Message: parameter.Hint})
		}
	}
	return blockingErrors, warnings
}

// describeDryRunError turns a dry run failure into a message the calling model can act on.
// A dry run aborts on causes the per field hints cannot express, and the raw error alone rarely
// tells the caller what to change.
func describeDryRunError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "was not given in array"), strings.Contains(message, "contains non-string value"):
		return fmt.Sprintf("a parameter value has the wrong JSON type: %v\nCheck valueJSONType in the parameters list. A \"string[]\" parameter needs a JSON array of strings, not a single string.", err)
	case strings.Contains(message, "default credentials"), strings.Contains(message, "oauth2"), strings.Contains(message, "credentials"):
		return fmt.Sprintf("the dry run could not authenticate to Google Cloud: %v\nRun \"gcloud auth application-default login\" and try again.", err)
	case strings.Contains(message, context.Canceled.Error()):
		// The dry run graph runs concurrently, so a task cancelled by a sibling's failure can be
		// the error that surfaces. The root cause is only in the KHI log on stderr.
		return fmt.Sprintf("the dry run was cancelled because another task in the graph failed: %v\nThe underlying error is in the KHI server log on stderr. Check the credentials and the parameter value types, then try again.", err)
	default:
		return fmt.Sprintf("the dry run failed: %v", err)
	}
}

var _ mcp.Tool = (*prepareJobCommandTool)(nil)

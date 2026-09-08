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

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
)

// featureItem is a single entry of the khi_list_features result.
type featureItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	// EnabledByDefault marks the features KHI turns on when the caller passes no feature list.
	EnabledByDefault bool `json:"enabledByDefault"`
}

// listFeaturesResult is the khi_list_features result.
type listFeaturesResult struct {
	InspectionType string        `json:"inspectionType"`
	Features       []featureItem `json:"features"`
	Note           string        `json:"note"`
	NextStep       string        `json:"nextStep"`
}

// listFeaturesArguments are the khi_list_features arguments.
type listFeaturesArguments struct {
	InspectionType string `json:"inspectionType"`
}

// listFeaturesTool lists the log sources an inspection type can collect.
type listFeaturesTool struct {
	deps Dependencies
}

func (t *listFeaturesTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:  "khi_list_features",
		Title: "List KHI features for an inspection type",
		Description: "Lists the log sources an inspection type can collect. Each description says which " +
			"logs the feature reads and which kind of problem it helps investigate, so use them to pick the " +
			"features matching the incident. Needs no credentials.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "inspectionType": {
      "type": "string",
      "description": "An id returned by khi_list_inspection_types, for example \"gcp-gke\"."
    }
  },
  "required": ["inspectionType"],
  "additionalProperties": false
}`),
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
}

func (t *listFeaturesTool) Call(ctx context.Context, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	args := listFeaturesArguments{}
	if err := decodeArguments(arguments, &args); err != nil {
		return mcp.ErrorResult("%v", err), nil
	}
	if args.InspectionType == "" {
		return mcp.ErrorResult("inspectionType is required. %s", t.knownInspectionTypesHint()), nil
	}

	inspectionID, err := t.deps.InspectionServer.CreateInspection(args.InspectionType)
	if err != nil {
		return mcp.ErrorResult("unknown inspection type %q. %s", args.InspectionType, t.knownInspectionTypesHint()), nil
	}
	defer t.deps.InspectionServer.DeleteInspection(inspectionID)

	features, err := t.deps.InspectionServer.GetInspection(inspectionID).FeatureList()
	if err != nil {
		return mcp.ErrorResult("failed to list the features of %q: %v", args.InspectionType, err), nil
	}
	items := make([]featureItem, 0, len(features))
	for _, feature := range features {
		items = append(items, featureItem{
			ID:               feature.Id,
			Label:            feature.Label,
			Description:      feature.Description,
			EnabledByDefault: feature.Enabled,
		})
	}
	return mcp.JSONResult(listFeaturesResult{
		InspectionType: args.InspectionType,
		Features:       items,
		Note: "Feature ids must be copied verbatim, including the part after '#'. Pass [\"ALL\"] to " +
			"khi_prepare_job_command to enable every feature, or omit the features argument to use the " +
			"ones marked enabledByDefault.",
		NextStep: "Call khi_prepare_job_command with the inspection type, the chosen features and an empty " +
			"values object to discover the parameters.",
	})
}

// knownInspectionTypesHint lists the valid inspection type ids, so a wrong id is self correcting.
func (t *listFeaturesTool) knownInspectionTypesHint() string {
	ids := []string{}
	for _, inspectionType := range t.deps.InspectionServer.GetAllInspectionTypes() {
		ids = append(ids, inspectionType.Id)
	}
	return fmt.Sprintf("Valid inspection types are: %s.", strings.Join(ids, ", "))
}

var _ mcp.Tool = (*listFeaturesTool)(nil)

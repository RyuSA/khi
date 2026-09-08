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

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
)

// ServerInstructions is returned in the initialize result. Harnesses put it into the model
// context without spending a tool call, so it states the loop as briefly as it can.
const ServerInstructions = `Kubernetes History Inspector (KHI) reconstructs an interactive timeline of a cluster from its logs.

This server does not run inspections. It produces the "khi --job-mode" command line for you to run yourself, and it is the authoritative source of the parameter schema: the parameter set changes with the inspection type, the enabled features and live cloud state, so never guess parameter names.

Loop: khi_list_inspection_types -> khi_list_features -> khi_prepare_job_command with empty values to discover the parameters -> fill them in and call it again until "ready" is true -> run the returned "jobCommand". Call khi_usage_guide for the full explanation.`

// usageGuideText is the long form guidance returned by the khi_usage_guide tool.
const usageGuideText = `# Using KHI from an agent

KHI (Kubernetes History Inspector) collects Kubernetes and Google Cloud logs for a time range and
builds a ".khi" file: an interactive timeline of every resource revision and log entry in that
window. It is a troubleshooting tool for answering "what happened to this cluster, and when".

## What this MCP server does and does not do

It **describes** KHI's job mode and **generates** the command line for it. It does **not** execute
the inspection. You run the returned command yourself, then open the resulting ".khi" file in the
KHI web UI (start it by running the KHI binary with no flags, then upload the file).

## The loop

1. **khi_list_inspection_types** - choose the platform the logs come from, for example "gcp-gke"
   for Google Kubernetes Engine or "oss-kubernetes-from-files" for log files you already have.
2. **khi_list_features** - choose which log sources to collect. Each feature says which logs it
   reads and what it lets you investigate. Pass ["ALL"] if you are unsure; it costs more query
   time but misses nothing.
3. **khi_prepare_job_command** with "values": {} - returns every parameter for that combination,
   each with its type, description, default and a "hint" explaining what is wrong with the current
   value. Parameters are not fixed: they depend on the inspection type, on the enabled features and
   on live cloud state such as which clusters exist in the project.
4. Fill in "values" and call it again. Repeat until "ready" is true. The "blockingErrors" array
   tells you exactly which parameters still need attention.
5. Run the returned "jobCommand" with your shell. It writes the ".khi" file to the path in
   "exportDestination".

## Parameter rules

- Parameter IDs are fully qualified task reference IDs such as
  "cloud.google.com/common/input-project-id". They are not short names like "projectId". Always
  copy them from khi_prepare_job_command output.
- Match "valueJSONType" exactly. A "string" parameter takes a JSON string; a "string[]" parameter
  takes a JSON array of strings. Passing a string where an array is expected fails the whole call.
- The time range is expressed as an end time plus a duration, not as a start and end pair:
  "cloud.google.com/common/input-end-time" is an RFC3339 timestamp and
  "cloud.google.com/common/input-duration" is a Go duration such as "3h" or "90m". To investigate
  an incident, set the end time slightly after the incident and the duration wide enough to cover
  what led up to it.
- Set parameters often accept alias values beginning with "@", such as "@default" or "@any", and
  negations beginning with "-". The options list in the output shows what is accepted.
- File parameters take a **local filesystem path** on the machine that will run the generated
  command, not an upload token.

## Prerequisites

The "gcp-*" inspection types read Google Cloud Logging and need Application Default Credentials.
If a call fails with a credentials error, run "gcloud auth application-default login". Preparing a
command performs a dry run that queries live cloud APIs for autocompletion and log volume
estimates, so it needs network access and is not instantaneous.`

// usageGuideTool returns the long form guidance describing the tool loop.
type usageGuideTool struct{}

func (t *usageGuideTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:  "khi_usage_guide",
		Title: "How to use KHI from an agent",
		Description: "Explains what KHI is, what a .khi file contains, and the exact sequence of tool " +
			"calls that turns an incident report into a runnable KHI job mode command. Read this first " +
			"when you have not used these tools before, or when a call fails in a way you do not understand.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
}

func (t *usageGuideTool) Call(ctx context.Context, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	return mcp.TextResult(usageGuideText), nil
}

var _ mcp.Tool = (*usageGuideTool)(nil)

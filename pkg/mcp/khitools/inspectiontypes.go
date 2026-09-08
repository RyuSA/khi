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

// inspectionTypeItem is a single entry of the khi_list_inspection_types result.
type inspectionTypeItem struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// listInspectionTypesResult is the khi_list_inspection_types result.
type listInspectionTypesResult struct {
	InspectionTypes []inspectionTypeItem `json:"inspectionTypes"`
	NextStep        string               `json:"nextStep"`
}

// listInspectionTypesTool lists the platforms KHI can gather logs from.
type listInspectionTypesTool struct {
	deps Dependencies
}

func (t *listInspectionTypesTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:  "khi_list_inspection_types",
		Title: "List KHI inspection types",
		Description: "Lists the platforms KHI can gather logs from, such as GKE, Cloud Composer or plain " +
			"Kubernetes log files. Call this first: the returned id selects which parameters and features " +
			"exist for everything that follows. Needs no credentials.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
}

func (t *listInspectionTypesTool) Call(ctx context.Context, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	types := t.deps.InspectionServer.GetAllInspectionTypes()
	items := make([]inspectionTypeItem, 0, len(types))
	for _, inspectionType := range types {
		items = append(items, inspectionTypeItem{
			ID:          inspectionType.Id,
			Name:        inspectionType.Name,
			Description: inspectionType.Description,
			Labels:      inspectionType.Labels,
		})
	}
	return mcp.JSONResult(listInspectionTypesResult{
		InspectionTypes: items,
		NextStep:        "Pass the chosen id to khi_list_features to see which log sources it can collect.",
	})
}

var _ mcp.Tool = (*listInspectionTypesTool)(nil)

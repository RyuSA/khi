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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListInspectionTypes(t *testing.T) {
	tool := &listInspectionTypesTool{deps: newFakeDependencies(t)}
	result, err := tool.Call(context.Background(), nil)
	if err != nil {
		t.Fatalf("Call returned an unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError = true, want false. content: %s", result.Content[0].Text)
	}
	decoded := listInspectionTypesResult{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &decoded); err != nil {
		t.Fatalf("the tool result is not decodable JSON: %v", err)
	}
	want := []inspectionTypeItem{{
		ID:          fakeInspectionTypeID,
		Name:        "Fake Inspection Type",
		Description: "An inspection type used by the KHI MCP tool tests.",
		Labels:      map[string]string{fakeInspectionTypeLabelKey: fakeInspectionTypeLabelValue},
	}}
	if diff := cmp.Diff(want, decoded.InspectionTypes); diff != "" {
		t.Errorf("InspectionTypes mismatch (-want +got):\n%s", diff)
	}
	if decoded.NextStep == "" {
		t.Error("NextStep is empty, want guidance towards the next tool")
	}
}

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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListFeatures(t *testing.T) {
	deps := newFakeDependencies(t)
	tool := &listFeaturesTool{deps: deps}
	result, err := tool.Call(context.Background(), json.RawMessage(`{"inspectionType":"`+fakeInspectionTypeID+`"}`))
	if err != nil {
		t.Fatalf("Call returned an unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError = true, want false. content: %s", result.Content[0].Text)
	}
	decoded := listFeaturesResult{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &decoded); err != nil {
		t.Fatalf("the tool result is not decodable JSON: %v", err)
	}
	want := []featureItem{{
		ID:               fakeFeatureID,
		Label:            "Fake feature",
		Description:      "Gathers nothing at all.",
		EnabledByDefault: true,
	}}
	if diff := cmp.Diff(want, decoded.Features); diff != "" {
		t.Errorf("Features mismatch (-want +got):\n%s", diff)
	}

	// The tool must not leave the inspection it created behind.
	if runners := deps.InspectionServer.GetAllRunners(); len(runners) != 0 {
		t.Errorf("got %d inspection runners left behind, want 0", len(runners))
	}
}

func TestListFeaturesRejectsBadArguments(t *testing.T) {
	testCases := []struct {
		name                string
		arguments           string
		wantMessageContains string
	}{
		{name: "missing inspection type", arguments: `{}`, wantMessageContains: "inspectionType is required"},
		{
			name:                "unknown inspection type lists the valid ones",
			arguments:           `{"inspectionType":"oss-log"}`,
			wantMessageContains: fakeInspectionTypeID,
		},
		{
			name:                "a misspelled argument name is rejected rather than ignored",
			arguments:           `{"inspection_type":"` + fakeInspectionTypeID + `"}`,
			wantMessageContains: `unknown field "inspection_type"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &listFeaturesTool{deps: newFakeDependencies(t)}
			result, err := tool.Call(context.Background(), json.RawMessage(tc.arguments))
			if err != nil {
				t.Fatalf("Call returned an unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("IsError = false, want true. content: %s", result.Content[0].Text)
			}
			if !strings.Contains(result.Content[0].Text, tc.wantMessageContains) {
				t.Errorf("message %q does not contain %q", result.Content[0].Text, tc.wantMessageContains)
			}
		})
	}
}

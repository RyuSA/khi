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

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	"github.com/google/go-cmp/cmp"
)

// callPrepare runs khi_prepare_job_command with the given arguments and decodes the result.
func callPrepare(t *testing.T, deps Dependencies, arguments map[string]any) (*mcp.CallToolResult, prepareJobCommandResult) {
	t.Helper()
	encoded, err := json.Marshal(arguments)
	if err != nil {
		t.Fatal(err)
	}
	tool := &prepareJobCommandTool{deps: deps}
	result, err := tool.Call(context.Background(), encoded)
	if err != nil {
		t.Fatalf("Call returned an unexpected error: %v", err)
	}
	decoded := prepareJobCommandResult{}
	if !result.IsError {
		if err := json.Unmarshal([]byte(result.Content[0].Text), &decoded); err != nil {
			t.Fatalf("the tool result is not decodable JSON %q: %v", result.Content[0].Text, err)
		}
	}
	return result, decoded
}

func TestPrepareJobCommandReportsMissingValuesAsBlockingErrors(t *testing.T) {
	deps := newFakeDependencies(t)
	result, decoded := callPrepare(t, deps, map[string]any{
		"inspectionType": fakeInspectionTypeID,
		"features":       []string{"ALL"},
		"values":         map[string]any{},
	})

	// Discovery must succeed even though the values are unusable, otherwise the caller never
	// learns which parameters exist.
	if result.IsError {
		t.Fatalf("IsError = true, want false. content: %s", result.Content[0].Text)
	}
	if decoded.Ready {
		t.Errorf("Ready = true, want false while a required parameter is unset")
	}
	if decoded.JobCommand != "" {
		t.Errorf("JobCommand = %q, want it withheld until Ready is true", decoded.JobCommand)
	}
	want := []parameterProblem{{ParameterID: fakeClusterFieldID, Message: "Cluster name must not be empty"}}
	if diff := cmp.Diff(want, decoded.BlockingErrors); diff != "" {
		t.Errorf("BlockingErrors mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{fakeFeatureID}, decoded.EnabledFeatures); diff != "" {
		t.Errorf("EnabledFeatures mismatch (-want +got):\n%s", diff)
	}
}

func TestPrepareJobCommandDescribesEveryParameter(t *testing.T) {
	deps := newFakeDependencies(t)
	_, decoded := callPrepare(t, deps, map[string]any{
		"inspectionType": fakeInspectionTypeID,
		"features":       []string{"ALL"},
		"values":         map[string]any{},
	})

	byID := map[string]parameterItem{}
	for _, parameter := range decoded.Parameters {
		assertHintTypeIsKnown(t, parameter.HintType)
		byID[parameter.ID] = parameter
	}

	testCases := []struct {
		name              string
		parameterID       string
		wantType          string
		wantValueJSONType string
		wantOptions       []string
	}{
		{
			name:              "a text parameter is reported as a JSON string",
			parameterID:       fakeClusterFieldID,
			wantType:          "text",
			wantValueJSONType: "string",
		},
		{
			name:              "a set parameter is reported as a JSON string array with its options",
			parameterID:       fakeKindsFieldID,
			wantType:          "set",
			wantValueJSONType: "string[]",
			wantOptions:       []string{"pods", "deployments"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parameter, found := byID[tc.parameterID]
			if !found {
				t.Fatalf("parameter %s is missing from the result", tc.parameterID)
			}
			if diff := cmp.Diff(tc.wantType, parameter.Type); diff != "" {
				t.Errorf("Type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantValueJSONType, parameter.ValueJSONType); diff != "" {
				t.Errorf("ValueJSONType mismatch (-want +got):\n%s", diff)
			}
			if tc.wantOptions != nil {
				if diff := cmp.Diff(tc.wantOptions, parameter.Options); diff != "" {
					t.Errorf("Options mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestPrepareJobCommandBuildsTheCommandOnceTheValuesAreAcceptable(t *testing.T) {
	deps := newFakeDependencies(t)
	_, decoded := callPrepare(t, deps, map[string]any{
		"inspectionType":    fakeInspectionTypeID,
		"features":          []string{"ALL"},
		"values":            fakeValuesWithCluster("my-cluster"),
		"exportDestination": "/tmp/incident.khi",
	})

	if !decoded.Ready {
		t.Fatalf("Ready = false, want true. blockingErrors: %v", decoded.BlockingErrors)
	}
	want := `/usr/local/bin/khi \
  --job-mode \
  --job-inspection-type="fake-inspection-type" \
  --job-inspection-features="khi.example.com/fake/feature#default" \
  --job-inspection-values='{
  "khi.example.com/fake/cluster-name": "my-cluster"
}' \
  --job-export-destination="/tmp/incident.khi"`
	if diff := cmp.Diff(want, decoded.JobCommand); diff != "" {
		t.Errorf("JobCommand mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/tmp/incident.khi", decoded.ExportDestination); diff != "" {
		t.Errorf("ExportDestination mismatch (-want +got):\n%s", diff)
	}
}

func TestPrepareJobCommandDefaultsTheExportDestination(t *testing.T) {
	deps := newFakeDependencies(t)
	_, decoded := callPrepare(t, deps, map[string]any{
		"inspectionType": fakeInspectionTypeID,
		"values":         fakeValuesWithCluster("my-cluster"),
	})
	if !decoded.Ready {
		t.Fatalf("Ready = false, want true. blockingErrors: %v", decoded.BlockingErrors)
	}
	if !strings.Contains(decoded.JobCommand, `--job-export-destination="`+DefaultExportDestination+`"`) {
		t.Errorf("JobCommand does not carry the default export destination:\n%s", decoded.JobCommand)
	}
}

func TestPrepareJobCommandRejectsBadArguments(t *testing.T) {
	testCases := []struct {
		name string
		// arguments is the raw arguments object.
		arguments string
		// wantMessageContains is a substring the failure message must carry so that the calling
		// model can work out what to change.
		wantMessageContains string
	}{
		{
			name:                "missing inspection type",
			arguments:           `{}`,
			wantMessageContains: "inspectionType is required",
		},
		{
			name:                "unknown inspection type lists the valid ones",
			arguments:           `{"inspectionType":"gke-basic"}`,
			wantMessageContains: fakeInspectionTypeID,
		},
		{
			name:                "unknown feature points at khi_list_features",
			arguments:           `{"inspectionType":"` + fakeInspectionTypeID + `","features":["nope"]}`,
			wantMessageContains: "khi_list_features",
		},
		{
			name:                "a misspelled argument name is rejected rather than ignored",
			arguments:           `{"inspection_type":"` + fakeInspectionTypeID + `"}`,
			wantMessageContains: `unknown field "inspection_type"`,
		},
		{
			name:                "a string where an array is expected explains the JSON type",
			arguments:           `{"inspectionType":"` + fakeInspectionTypeID + `","values":{"` + fakeKindsFieldID + `":"pods"}}`,
			wantMessageContains: "valueJSONType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &prepareJobCommandTool{deps: newFakeDependencies(t)}
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

func TestPrepareJobCommandLeavesNoInspectionBehind(t *testing.T) {
	// The tools are stateless on purpose: InspectionTaskServer never evicts runners, so a tool
	// that forgot to delete its inspection would leak one per call.
	deps := newFakeDependencies(t)
	for range 3 {
		callPrepare(t, deps, map[string]any{
			"inspectionType": fakeInspectionTypeID,
			"values":         fakeValuesWithCluster("my-cluster"),
		})
	}
	if runners := deps.InspectionServer.GetAllRunners(); len(runners) != 0 {
		t.Errorf("got %d inspection runners left behind, want 0", len(runners))
	}
}

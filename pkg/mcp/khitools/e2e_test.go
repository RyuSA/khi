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
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	mcpstdio "github.com/GoogleCloudPlatform/khi/pkg/mcp/stdio"
	"github.com/google/go-cmp/cmp"
)

// runSession drives a full stdio session and returns the decoded responses keyed by request id.
func runSession(t *testing.T, deps Dependencies, requests []string) map[float64]map[string]any {
	t.Helper()
	tools, err := NewTools(deps)
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	server, err := mcp.NewServer(mcp.Implementation{Name: "khi", Version: "test"}, ServerInstructions, tools)
	if err != nil {
		t.Fatalf("NewServer returned an unexpected error: %v", err)
	}

	input := strings.Join(requests, "\n") + "\n"
	output := &bytes.Buffer{}
	if err := mcpstdio.Serve(context.Background(), strings.NewReader(input), output, server); err != nil {
		t.Fatalf("Serve returned an unexpected error: %v", err)
	}

	responses := map[float64]map[string]any{}
	for _, line := range strings.Split(strings.TrimRight(output.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		message := map[string]any{}
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatalf("the transport wrote a line that is not valid JSON %q: %v", line, err)
		}
		id, isNumber := message["id"].(float64)
		if !isNumber {
			t.Fatalf("a response carries a non numeric id: %v", message)
		}
		responses[id] = message
	}
	return responses
}

// toolPayload decodes the JSON document a tool returned in its single text content block.
func toolPayload(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, isMap := response["result"].(map[string]any)
	if !isMap {
		t.Fatalf("the response carries no result: %v", response)
	}
	if isError, _ := result["isError"].(bool); isError {
		t.Fatalf("the tool reported a failure: %v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(content["text"].(string)), &payload); err != nil {
		t.Fatalf("the tool result is not decodable JSON: %v", err)
	}
	return payload
}

func TestEndToEndSessionOverStdio(t *testing.T) {
	// This exercises the whole path an agent harness takes: handshake, tool discovery, then the
	// discover, fix, build loop, all over the real newline delimited transport.
	deps := newFakeDependencies(t)
	responses := runSession(t, deps, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"khi_list_inspection_types","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"khi_list_features","arguments":{"inspectionType":"` + fakeInspectionTypeID + `"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"khi_prepare_job_command","arguments":{"inspectionType":"` + fakeInspectionTypeID + `","features":["ALL"],"values":{}}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"khi_prepare_job_command","arguments":{"inspectionType":"` + fakeInspectionTypeID + `","features":["ALL"],"values":{"` + fakeClusterFieldID + `":"my-cluster"}}}}`,
	})

	// The notification must not have produced a response, so exactly six replies are expected.
	if diff := cmp.Diff(6, len(responses)); diff != "" {
		t.Errorf("response count mismatch (-want +got):\n%s", diff)
	}

	types := toolPayload(t, responses[3])["inspectionTypes"].([]any)
	if diff := cmp.Diff(1, len(types)); diff != "" {
		t.Errorf("inspection type count mismatch (-want +got):\n%s", diff)
	}

	features := toolPayload(t, responses[4])["features"].([]any)
	if diff := cmp.Diff(fakeFeatureID, features[0].(map[string]any)["id"]); diff != "" {
		t.Errorf("feature id mismatch (-want +got):\n%s", diff)
	}

	discovery := toolPayload(t, responses[5])
	if discovery["ready"] != false {
		t.Errorf("ready = %v on the discovery call, want false", discovery["ready"])
	}
	if len(discovery["blockingErrors"].([]any)) == 0 {
		t.Error("blockingErrors is empty on the discovery call, want the unset parameter reported")
	}

	completed := toolPayload(t, responses[6])
	if completed["ready"] != true {
		t.Errorf("ready = %v after the values were supplied, want true", completed["ready"])
	}
	command, _ := completed["jobCommand"].(string)
	for _, fragment := range []string{"--job-mode", `--job-inspection-type="` + fakeInspectionTypeID + `"`, "my-cluster"} {
		if !strings.Contains(command, fragment) {
			t.Errorf("jobCommand does not contain %q:\n%s", fragment, command)
		}
	}
}

func TestEndToEndSessionWritesOnlyJSONToTheStream(t *testing.T) {
	// The stdio transport shares its stream with nothing else. A single non JSON line breaks the
	// client connection, so this asserts the property directly rather than by inspection.
	deps := newFakeDependencies(t)
	tools, err := NewTools(deps)
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	server, err := mcp.NewServer(mcp.Implementation{Name: "khi", Version: "test"}, ServerInstructions, tools)
	if err != nil {
		t.Fatalf("NewServer returned an unexpected error: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		// A tool call that fails, because a failure path is the likeliest source of stray output.
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"khi_list_features","arguments":{"inspectionType":"nope"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"does/not/exist"}`,
	}, "\n") + "\n"

	output := &bytes.Buffer{}
	if err := mcpstdio.Serve(context.Background(), strings.NewReader(input), output, server); err != nil {
		t.Fatalf("Serve returned an unexpected error: %v", err)
	}
	for _, line := range strings.Split(strings.TrimRight(output.String(), "\n"), "\n") {
		message := map[string]any{}
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Errorf("the transport wrote a non JSON line %q: %v", line, err)
		}
	}
}

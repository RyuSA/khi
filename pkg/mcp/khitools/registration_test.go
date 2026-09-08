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
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestNewToolsExposesTheToolsInTheIntendedOrder(t *testing.T) {
	tools, err := NewTools(newFakeDependencies(t))
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	names := []string{}
	for _, tool := range tools {
		names = append(names, tool.Definition().Name)
	}
	// Models tend to try tools in the order they are listed, so the order is the loop order.
	want := []string{"khi_usage_guide", "khi_list_inspection_types", "khi_list_features", "khi_prepare_job_command"}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("tool names mismatch (-want +got):\n%s", diff)
	}
}

func TestNewToolsRejectsIncompleteDependencies(t *testing.T) {
	testCases := []struct {
		name string
		deps Dependencies
	}{
		{name: "no inspection server", deps: Dependencies{BinaryPath: "/usr/local/bin/khi"}},
		{name: "no binary path", deps: Dependencies{InspectionServer: newFakeInspectionServer(t)}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewTools(tc.deps); err == nil {
				t.Error("NewTools returned nil error, want an error")
			}
		})
	}
}

func TestToolDefinitions(t *testing.T) {
	// The tool names, descriptions and input schemas are read by a model rather than by a human,
	// so they are effectively part of the prompt. Pinning them to a golden file makes any change
	// to the agent's instructions visible in review instead of silent.
	testutil.InitTestIO()
	tools, err := NewTools(newFakeDependencies(t))
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	definitions := []mcp.ToolDefinition{}
	for _, tool := range tools {
		definitions = append(definitions, tool.Definition())
	}
	encoded, err := json.MarshalIndent(definitions, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	testutil.VerifyWithGolden(t, "tools-list", string(encoded))
}

func TestServerInstructionsNameEveryTool(t *testing.T) {
	// The instructions are the only guidance a model gets without spending a tool call, so a tool
	// missing from them is effectively invisible until something else mentions it.
	tools, err := NewTools(newFakeDependencies(t))
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	for _, tool := range tools {
		name := tool.Definition().Name
		if !strings.Contains(ServerInstructions, name) {
			t.Errorf("ServerInstructions does not mention the tool %s", name)
		}
		if name == "khi_usage_guide" {
			// The guide is the document itself and has no reason to name itself in its body.
			continue
		}
		if !strings.Contains(usageGuideText, name) {
			t.Errorf("usageGuideText does not mention the tool %s", name)
		}
	}
}

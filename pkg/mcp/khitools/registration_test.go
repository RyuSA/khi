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
	want := []string{"khi_list_inspection_types", "khi_list_features", "khi_prepare_job_command"}
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
	// The instructions are the only place this server explains itself, and the only guidance a
	// model gets without spending a tool call. A tool missing from them is effectively invisible
	// until the model reads tools/list closely.
	tools, err := NewTools(newFakeDependencies(t))
	if err != nil {
		t.Fatalf("NewTools returned an unexpected error: %v", err)
	}
	for _, tool := range tools {
		name := tool.Definition().Name
		if !strings.Contains(ServerInstructions, name) {
			t.Errorf("ServerInstructions does not mention the tool %s", name)
		}
	}
}

func TestServerInstructionsLoopFitsInThePrefixBudget(t *testing.T) {
	// Codex treats the leading InstructionsPrefixBudget characters as the guidance it has while
	// deciding how to use the server. The sentence describing the loop is the single most useful
	// thing in the text, so it has to finish inside that prefix rather than being cut in half.
	loopEnd := strings.Index(ServerInstructions, instructionsLoopSentenceEnd)
	if loopEnd < 0 {
		t.Fatalf("ServerInstructions no longer contains the loop sentence ending %q", instructionsLoopSentenceEnd)
	}
	loopEnd += len(instructionsLoopSentenceEnd)

	if loopEnd > InstructionsPrefixBudget {
		t.Errorf("the loop sentence ends at character %d, past the %d character prefix budget.\nShorten the text before it so the loop stays self-contained:\n%s",
			loopEnd, InstructionsPrefixBudget, ServerInstructions[:InstructionsPrefixBudget])
	}
}

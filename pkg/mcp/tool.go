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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool is a single tool exposed through tools/list and tools/call.
type Tool interface {
	// Definition returns the metadata advertised through tools/list.
	//
	// The description and the input schema are read by the model, not by a human, so they are
	// effectively part of the prompt. Treat a change to them as a behavioural change.
	Definition() ToolDefinition
	// Call executes the tool with the raw arguments object sent by the client.
	// A returned error is reported to the client as a tool execution error, so that the calling
	// model can read the message and retry with corrected arguments.
	Call(ctx context.Context, arguments json.RawMessage) (*CallToolResult, error)
}

// TextResult builds a successful tool result carrying a single text block.
func TextResult(text string) *CallToolResult {
	return &CallToolResult{Content: []Content{{Type: "text", Text: text}}}
}

// ErrorResult builds a failed tool result carrying a single text block.
// The failure reaches the model as tool output rather than as a protocol error.
func ErrorResult(format string, args ...any) *CallToolResult {
	return &CallToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf(format, args...)}},
		IsError: true,
	}
}

// JSONResult builds a successful tool result carrying the indented JSON encoding of value.
func JSONResult(value any) (*CallToolResult, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode the tool result: %w", err)
	}
	return TextResult(string(encoded)), nil
}

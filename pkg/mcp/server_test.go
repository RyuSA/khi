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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// fakeTool is a Tool returning a canned result, used to exercise the dispatch paths.
type fakeTool struct {
	name    string
	err     error
	result  *CallToolResult
	lastArg string
}

func (f *fakeTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        f.name,
		Description: "fake tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
	}
}

func (f *fakeTool) Call(ctx context.Context, arguments json.RawMessage) (*CallToolResult, error) {
	f.lastArg = string(arguments)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

var _ Tool = (*fakeTool)(nil)

func newTestServer(t *testing.T, tools ...Tool) *Server {
	t.Helper()
	server, err := NewServer(Implementation{Name: "khi", Version: "test"}, "instructions", tools)
	if err != nil {
		t.Fatalf("NewServer returned an unexpected error: %v", err)
	}
	return server
}

func TestServerHandleMessage(t *testing.T) {
	testCases := []struct {
		name string
		// input is the raw inbound message.
		input string
		// want is the expected response decoded as JSON. A nil want means no response at all.
		want map[string]any
	}{
		{
			name:  "initialize echoes a supported protocol version",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"c","version":"0"}}}`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(1),
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "khi", "version": "test"},
					"instructions":    "instructions",
				},
			},
		},
		{
			name:  "initialize falls back for an unknown protocol version instead of failing",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01","clientInfo":{"name":"c","version":"0"}}}`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(1),
				"result": map[string]any{
					"protocolVersion": ProtocolVersion,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "khi", "version": "test"},
					"instructions":    "instructions",
				},
			},
		},
		{
			name:  "initialized notification gets no response",
			input: `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			want:  nil,
		},
		{
			name:  "an unknown notification is tolerated silently",
			input: `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`,
			want:  nil,
		},
		{
			name:  "ping returns an empty result",
			input: `{"jsonrpc":"2.0","id":"a","method":"ping"}`,
			want:  map[string]any{"jsonrpc": "2.0", "id": "a", "result": map[string]any{}},
		},
		{
			name:  "an unknown method is a JSON-RPC method not found error",
			input: `{"jsonrpc":"2.0","id":7,"method":"does/not/exist"}`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(7),
				"error":   map[string]any{"code": float64(CodeMethodNotFound), "message": "method not found: does/not/exist"},
			},
		},
		{
			name:  "an unknown tool is an invalid params error",
			input: `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(8),
				"error":   map[string]any{"code": float64(CodeInvalidParams), "message": "unknown tool: nope"},
			},
		},
		{
			name:  "malformed JSON is a parse error with a null id",
			input: `{"jsonrpc":"2.0",`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      nil,
				"error":   map[string]any{"code": float64(CodeParseError), "message": "failed to parse the message: unexpected end of JSON input"},
			},
		},
		{
			name:  "a batch request is rejected",
			input: `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      nil,
				"error":   map[string]any{"code": float64(CodeInvalidRequest), "message": "batch requests are not supported"},
			},
		},
		{
			name:  "resources/list answers with an empty list",
			input: `{"jsonrpc":"2.0","id":9,"method":"resources/list"}`,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(9),
				"result":  map[string]any{"resources": []any{}},
			},
		},
		{
			name:  "a blank line produces no response",
			input: "   ",
			want:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, &fakeTool{name: "fake", result: TextResult("ok")})
			got := server.HandleMessage(context.Background(), []byte(tc.input))
			if tc.want == nil {
				if got != nil {
					t.Fatalf("HandleMessage() = %s, want no response", string(got))
				}
				return
			}
			decoded := map[string]any{}
			if err := json.Unmarshal(got, &decoded); err != nil {
				t.Fatalf("HandleMessage() returned undecodable JSON %q: %v", string(got), err)
			}
			if diff := cmp.Diff(tc.want, decoded); diff != "" {
				t.Errorf("HandleMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServerHandleMessageEchoesTheIDVerbatim(t *testing.T) {
	testCases := []struct {
		name string
		id   string
	}{
		{name: "string id", id: `"request-1"`},
		{name: "small integer id", id: `1`},
		{name: "integer id beyond float64 precision", id: `9007199254740993`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t)
			got := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":`+tc.id+`,"method":"ping"}`))
			envelope := struct {
				ID json.RawMessage `json:"id"`
			}{}
			if err := json.Unmarshal(got, &envelope); err != nil {
				t.Fatalf("HandleMessage() returned undecodable JSON %q: %v", string(got), err)
			}
			if diff := cmp.Diff(tc.id, string(envelope.ID)); diff != "" {
				t.Errorf("response id mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServerHandleMessageReportsToolFailureAsToolOutput(t *testing.T) {
	// A failing tool must not become a JSON-RPC error, otherwise the calling model never sees
	// the message and cannot correct its arguments.
	server := newTestServer(t, &fakeTool{name: "fake", err: errors.New("bad argument")})
	got := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fake","arguments":{}}}`))

	response := struct {
		Result *CallToolResult `json:"result"`
		Error  *Error          `json:"error"`
	}{}
	if err := json.Unmarshal(got, &response); err != nil {
		t.Fatalf("HandleMessage() returned undecodable JSON %q: %v", string(got), err)
	}
	if response.Error != nil {
		t.Fatalf("got a JSON-RPC error %v, want a tool result", response.Error)
	}
	if !response.Result.IsError {
		t.Errorf("IsError = false, want true")
	}
	want := []Content{{Type: "text", Text: "fake failed: bad argument"}}
	if diff := cmp.Diff(want, response.Result.Content); diff != "" {
		t.Errorf("content mismatch (-want +got):\n%s", diff)
	}
}

func TestServerListsRegisteredTools(t *testing.T) {
	server := newTestServer(t, &fakeTool{name: "b"}, &fakeTool{name: "a"})
	got := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))

	response := struct {
		Result ListToolsResult `json:"result"`
	}{}
	if err := json.Unmarshal(got, &response); err != nil {
		t.Fatalf("HandleMessage() returned undecodable JSON %q: %v", string(got), err)
	}
	names := []string{}
	for _, definition := range response.Result.Tools {
		names = append(names, definition.Name)
	}
	// Registration order is preserved so that the golden test over the definitions is stable.
	if diff := cmp.Diff([]string{"b", "a"}, names); diff != "" {
		t.Errorf("tool names mismatch (-want +got):\n%s", diff)
	}
}

func TestNewServerRejectsDuplicatedToolNames(t *testing.T) {
	_, err := NewServer(Implementation{Name: "khi", Version: "test"}, "", []Tool{
		&fakeTool{name: "same"},
		&fakeTool{name: "same"},
	})
	if err == nil {
		t.Fatal("NewServer returned nil error, want an error for the duplicated tool name")
	}
}

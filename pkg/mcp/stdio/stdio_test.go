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

package mcpstdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	"github.com/google/go-cmp/cmp"
)

// echoTool returns the arguments it received so tests can assert a large payload survived framing.
type echoTool struct{}

func (e *echoTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "echo",
		Description: "echoes the size of the given arguments",
		InputSchema: json.RawMessage(`{"type":"object","additionalProperties":true}`),
	}
}

func (e *echoTool) Call(ctx context.Context, arguments json.RawMessage) (*mcp.CallToolResult, error) {
	return mcp.TextResult(fmt.Sprintf("%d", len(arguments))), nil
}

var _ mcp.Tool = (*echoTool)(nil)

func newTestServer(t *testing.T) *mcp.Server {
	t.Helper()
	server, err := mcp.NewServer(mcp.Implementation{Name: "khi", Version: "test"}, "", []mcp.Tool{&echoTool{}})
	if err != nil {
		t.Fatalf("NewServer returned an unexpected error: %v", err)
	}
	return server
}

// decodeLines decodes every non empty line of the output into a generic map.
func decodeLines(t *testing.T, output string) []map[string]any {
	t.Helper()
	decoded := []map[string]any{}
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if line == "" {
			continue
		}
		message := map[string]any{}
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatalf("output line %q is not valid JSON: %v", line, err)
		}
		decoded = append(decoded, message)
	}
	return decoded
}

func TestServeFramesResponses(t *testing.T) {
	testCases := []struct {
		name string
		// input is the whole stdin content.
		input string
		// wantIDs are the ids of the expected responses, in order. Notifications produce none.
		wantIDs []any
	}{
		{
			name: "a full handshake answers only the requests",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"c","version":"0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
`,
			wantIDs: []any{float64(1), float64(2)},
		},
		{
			name: "blank lines between messages are skipped",
			input: `{"jsonrpc":"2.0","id":1,"method":"ping"}

{"jsonrpc":"2.0","id":2,"method":"ping"}
`,
			wantIDs: []any{float64(1), float64(2)},
		},
		{
			name:    "a final message without a trailing newline is still handled",
			input:   `{"jsonrpc":"2.0","id":1,"method":"ping"}`,
			wantIDs: []any{float64(1)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			if err := Serve(context.Background(), strings.NewReader(tc.input), output, newTestServer(t)); err != nil {
				t.Fatalf("Serve returned an unexpected error: %v", err)
			}
			gotIDs := []any{}
			for _, message := range decodeLines(t, output.String()) {
				gotIDs = append(gotIDs, message["id"])
			}
			if diff := cmp.Diff(tc.wantIDs, gotIDs); diff != "" {
				t.Errorf("response ids mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServeHandlesAMessageLargerThanTheReadBuffer(t *testing.T) {
	// bufio.Scanner would fail on a line above its own limit, which is why the transport reads
	// with bufio.Reader. There is no protocol level bound on the size of a tool argument.
	large := strings.Repeat("a", 4*readerBufferSize)
	arguments := map[string]any{"payload": large}
	encodedArguments, err := json.Marshal(arguments)
	if err != nil {
		t.Fatal(err)
	}
	input := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":%s}}`+"\n", string(encodedArguments))

	output := &bytes.Buffer{}
	if err := Serve(context.Background(), strings.NewReader(input), output, newTestServer(t)); err != nil {
		t.Fatalf("Serve returned an unexpected error: %v", err)
	}
	messages := decodeLines(t, output.String())
	if len(messages) != 1 {
		t.Fatalf("got %d responses, want 1", len(messages))
	}
	result, ok := messages[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("response has no result: %v", messages[0])
	}
	content := result["content"].([]any)[0].(map[string]any)
	if diff := cmp.Diff(fmt.Sprintf("%d", len(encodedArguments)), content["text"]); diff != "" {
		t.Errorf("echoed argument size mismatch (-want +got):\n%s", diff)
	}
}

func TestServeReturnsWhenTheContextIsCancelled(t *testing.T) {
	// A pipe that never delivers data nor EOF stands in for an idle client.
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, reader, &bytes.Buffer{}, newTestServer(t))
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve returned an unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the context was cancelled")
	}
}

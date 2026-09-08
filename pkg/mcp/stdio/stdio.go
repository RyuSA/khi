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

// Package mcpstdio serves an mcp.Server over the MCP stdio transport.
//
// The transport frames messages as newline delimited JSON. Nothing other than a JSON-RPC message
// may ever be written to the output stream, which is why KHI redirects all of its logging to
// stderr while this transport runs.
package mcpstdio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
)

// readerBufferSize is the initial size of the input buffer. bufio.Reader grows beyond it as
// needed, unlike bufio.Scanner which would fail on a line above its own limit. Inspection
// parameter payloads are small but there is no protocol level bound on them.
const readerBufferSize = 64 * 1024

// Serve reads newline delimited JSON-RPC messages from in, dispatches them through server and
// writes the responses to out. It returns when in reaches EOF or when ctx is cancelled.
//
// Messages are handled one at a time. Agent harnesses call tools sequentially, and serial
// handling keeps the responses ordered and avoids concurrent access to the inspection server.
func Serve(ctx context.Context, in io.Reader, out io.Writer, server *mcp.Server) error {
	reader := bufio.NewReaderSize(in, readerBufferSize)
	writer := bufio.NewWriter(out)
	done := make(chan error, 1)

	go func() {
		done <- serveLoop(ctx, reader, writer, server)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// The blocking read on the input stream cannot be interrupted, so the loop goroutine is
		// left to end with the process. Flushing here keeps any already written response intact.
		if flushErr := writer.Flush(); flushErr != nil {
			return flushErr
		}
		return nil
	}
}

func serveLoop(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer, server *mcp.Server) error {
	for {
		if ctx.Err() != nil {
			return writer.Flush()
		}
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			response := server.HandleMessage(ctx, line)
			if response != nil {
				if writeErr := writeMessage(writer, response); writeErr != nil {
					return writeErr
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return writer.Flush()
			}
			if flushErr := writer.Flush(); flushErr != nil {
				return flushErr
			}
			return fmt.Errorf("failed to read from the MCP input stream: %w", err)
		}
	}
}

// writeMessage writes one framed message and flushes it, so that the client sees the response
// without waiting for the buffer to fill.
func writeMessage(writer *bufio.Writer, message []byte) error {
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("failed to write to the MCP output stream: %w", err)
	}
	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write to the MCP output stream: %w", err)
	}
	return writer.Flush()
}

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Server dispatches MCP JSON-RPC messages to the registered tools.
//
// It performs no I/O and holds no per connection state, so every transport shares it unchanged.
type Server struct {
	info         Implementation
	instructions string
	tools        []Tool
	toolsByName  map[string]Tool
}

// NewServer creates a Server exposing the given tools.
// It returns an error when two tools share a name.
func NewServer(info Implementation, instructions string, tools []Tool) (*Server, error) {
	toolsByName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		name := tool.Definition().Name
		if name == "" {
			return nil, fmt.Errorf("a tool with an empty name was registered")
		}
		if _, duplicated := toolsByName[name]; duplicated {
			return nil, fmt.Errorf("duplicated tool registered: %s", name)
		}
		toolsByName[name] = tool
	}
	return &Server{
		info:         info,
		instructions: instructions,
		tools:        tools,
		toolsByName:  toolsByName,
	}, nil
}

// HandleMessage processes a single JSON-RPC message and returns the encoded response without a
// trailing newline. It returns nil when the message needs no response, which is the case for
// every notification.
func (s *Server) HandleMessage(ctx context.Context, message []byte) []byte {
	trimmed := bytes.TrimSpace(message)
	if len(trimmed) == 0 {
		return nil
	}
	// JSON-RPC batching was removed from MCP, and supporting it would complicate every transport
	// for no client that needs it.
	if trimmed[0] == '[' {
		return s.encodeError(nil, CodeInvalidRequest, "batch requests are not supported")
	}

	var request Request
	if err := json.Unmarshal(trimmed, &request); err != nil {
		return s.encodeError(nil, CodeParseError, fmt.Sprintf("failed to parse the message: %v", err))
	}
	if request.Method == "" {
		return s.encodeError(request.ID, CodeInvalidRequest, "the message has no method")
	}

	result, rpcError := s.dispatch(ctx, &request)
	if request.IsNotification() {
		// Notifications never get a response, not even when the handler failed.
		return nil
	}
	if rpcError != nil {
		return s.encodeResponse(&Response{JSONRPC: JSONRPCVersion, ID: request.ID, Error: rpcError})
	}
	return s.encodeResponse(&Response{JSONRPC: JSONRPCVersion, ID: request.ID, Result: result})
}

// dispatch routes one request to its handler. It returns either a result or a JSON-RPC error.
func (s *Server) dispatch(ctx context.Context, request *Request) (any, *Error) {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "ping":
		return emptyResult{}, nil
	case "tools/list":
		return s.handleListTools()
	case "tools/call":
		return s.handleCallTool(ctx, request)
	case "resources/list":
		return listResourcesResult{Resources: []struct{}{}}, nil
	case "resources/templates/list":
		return listResourceTemplatesResult{ResourceTemplates: []struct{}{}}, nil
	case "prompts/list":
		return listPromptsResult{Prompts: []struct{}{}}, nil
	default:
		if strings.HasPrefix(request.Method, "notifications/") {
			// Every notification is accepted and ignored. Failing on an unknown one would break
			// clients that send lifecycle notifications this server does not act on.
			return nil, nil
		}
		return nil, &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("method not found: %s", request.Method)}
	}
}

func (s *Server) handleInitialize(request *Request) (any, *Error) {
	params := InitializeParams{}
	if len(request.Params) > 0 {
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, &Error{Code: CodeInvalidParams, Message: fmt.Sprintf("failed to parse the initialize parameters: %v", err)}
		}
	}
	negotiated := ProtocolVersion
	if slices.Contains(SupportedProtocolVersions, params.ProtocolVersion) {
		negotiated = params.ProtocolVersion
	}
	return InitializeResult{
		ProtocolVersion: negotiated,
		Capabilities:    ServerCapabilities{Tools: &ToolsCapability{}},
		ServerInfo:      s.info,
		Instructions:    s.instructions,
	}, nil
}

func (s *Server) handleListTools() (any, *Error) {
	definitions := make([]ToolDefinition, 0, len(s.tools))
	for _, tool := range s.tools {
		definitions = append(definitions, tool.Definition())
	}
	return ListToolsResult{Tools: definitions}, nil
}

func (s *Server) handleCallTool(ctx context.Context, request *Request) (any, *Error) {
	params := CallToolParams{}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return nil, &Error{Code: CodeInvalidParams, Message: fmt.Sprintf("failed to parse the tools/call parameters: %v", err)}
	}
	tool, found := s.toolsByName[params.Name]
	if !found {
		return nil, &Error{Code: CodeInvalidParams, Message: fmt.Sprintf("unknown tool: %s", params.Name)}
	}
	result, err := tool.Call(ctx, params.Arguments)
	if err != nil {
		return ErrorResult("%s failed: %v", params.Name, err), nil
	}
	return result, nil
}

// encodeError builds an encoded error response for the given request ID.
// A nil id becomes a JSON null, which is what the specification requires when the ID is unknown.
func (s *Server) encodeError(id json.RawMessage, code int, message string) []byte {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return s.encodeResponse(&Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	})
}

// encodeResponse marshals a response into a single line of JSON.
func (s *Server) encodeResponse(response *Response) []byte {
	encoded, err := json.Marshal(response)
	if err != nil {
		// Falling back keeps the stream well formed even when a tool returned an unmarshalable
		// value, which would otherwise leave the client waiting forever for a response.
		fallback, fallbackErr := json.Marshal(&Response{
			JSONRPC: JSONRPCVersion,
			ID:      response.ID,
			Error:   &Error{Code: CodeInternalError, Message: fmt.Sprintf("failed to encode the response: %v", err)},
		})
		if fallbackErr != nil {
			return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"failed to encode the response"}}`)
		}
		return fallback
	}
	return encoded
}

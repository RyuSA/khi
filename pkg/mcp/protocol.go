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

// Package mcp implements the subset of the Model Context Protocol that KHI needs to expose
// its inspection parameter schema to agent harnesses.
//
// The package deliberately depends on the standard library only. It knows nothing about KHI
// types and nothing about transports: Server.HandleMessage maps one JSON-RPC message to at
// most one response message, which is the seam every transport is built on.
package mcp

import "encoding/json"

// ProtocolVersion is the MCP revision this server prefers when a client asks for something unknown.
const ProtocolVersion = "2025-06-18"

// SupportedProtocolVersions lists the MCP revisions this server can negotiate, newest first.
// A client asking for a version outside this list is answered with ProtocolVersion rather than
// an error, which is the negotiation path the specification describes.
var SupportedProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// JSONRPCVersion is the only JSON-RPC version accepted on the wire.
const JSONRPCVersion = "2.0"

// JSON-RPC 2.0 error codes used by this server.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request is an inbound JSON-RPC 2.0 request or notification.
//
// ID stays as raw JSON so the exact token is echoed back unchanged. Decoding it into an
// interface{} would turn a large integer ID into a float64 and change it on the way out.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether the message carries no ID and therefore expects no response.
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

// Response is an outbound JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Implementation identifies a party of the protocol.
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// InitializeParams are the parameters of the initialize request.
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      Implementation  `json:"clientInfo"`
}

// ToolsCapability declares the tool related capabilities of this server.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilities declares what this server supports.
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Instructions is surfaced by harnesses as part of the model context. It is the cheapest
	// place to explain the tool loop, because the model reads it without spending a tool call.
	Instructions string `json:"instructions,omitempty"`
}

// ToolAnnotations are optional behavioural hints shown alongside a tool.
type ToolAnnotations struct {
	ReadOnlyHint   bool `json:"readOnlyHint,omitempty"`
	IdempotentHint bool `json:"idempotentHint,omitempty"`
	OpenWorldHint  bool `json:"openWorldHint,omitempty"`
}

// ToolDefinition is the metadata advertised for a single tool through tools/list.
type ToolDefinition struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description"`
	InputSchema json.RawMessage  `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ListToolsResult is the result of the tools/list request.
type ListToolsResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// CallToolParams are the parameters of the tools/call request.
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Content is a single content block of a tool result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallToolResult is the result of the tools/call request.
//
// A tool that fails reports it through IsError instead of a JSON-RPC error, so that the calling
// model receives the message as tool output and can correct itself. JSON-RPC errors are reserved
// for protocol level problems the model cannot act on.
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// emptyResult is returned by requests whose result carries no fields, such as ping.
type emptyResult struct{}

// listResourcesResult answers resources/list. The resources capability is not advertised, but
// harnesses probing for it unconditionally get an empty list rather than a method-not-found error.
type listResourcesResult struct {
	Resources []struct{} `json:"resources"`
}

// listResourceTemplatesResult answers resources/templates/list for the same reason.
type listResourceTemplatesResult struct {
	ResourceTemplates []struct{} `json:"resourceTemplates"`
}

// listPromptsResult answers prompts/list for the same reason.
type listPromptsResult struct {
	Prompts []struct{} `json:"prompts"`
}

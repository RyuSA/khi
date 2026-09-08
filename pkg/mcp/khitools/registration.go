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
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
)

// NewTools builds the KHI tool set in the order the caller is meant to use them.
// tools/list preserves this order, and models tend to try the tools in the order they are listed.
func NewTools(deps Dependencies) ([]mcp.Tool, error) {
	if deps.InspectionServer == nil {
		return nil, fmt.Errorf("an inspection server is required to build the KHI MCP tools")
	}
	if deps.BinaryPath == "" {
		return nil, fmt.Errorf("a binary path is required to build the KHI MCP tools")
	}
	if deps.DefaultExportDestination == "" {
		deps.DefaultExportDestination = DefaultExportDestination
	}
	return []mcp.Tool{
		&usageGuideTool{},
		&listInspectionTypesTool{deps: deps},
		&listFeaturesTool{deps: deps},
		&prepareJobCommandTool{deps: deps},
	}, nil
}

// decodeArguments decodes a tools/call arguments object, rejecting unknown fields.
// A silently ignored typo in an argument name would look to the model like the argument had no
// effect, which is far harder to recover from than an explicit error.
func decodeArguments(arguments json.RawMessage, target any) error {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to parse the arguments: %v", err)
	}
	return nil
}

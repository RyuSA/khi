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

package parameters

import (
	"errors"

	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
)

// MCPModeFlagName is the command line flag name enabling the MCP server mode.
const MCPModeFlagName = "mcp-mode"

// MCPModeEnvKey is the environment variable enabling the MCP server mode.
const MCPModeEnvKey = "KHI_MCP_MODE"

var MCP *MCPParameters = &MCPParameters{}

// MCPParameters is the ParameterStore for the Model Context Protocol server mode.
type MCPParameters struct {
	// MCPMode
	// If this flag is set, KHI serves the Model Context Protocol over stdin/stdout and doesn't serve as a web server.
	MCPMode *bool
}

// Prepare implements ParameterStore.
func (m *MCPParameters) Prepare() error {
	m.MCPMode = flag.Bool(MCPModeFlagName, false, "If this flag is set, KHI serves the Model Context Protocol over stdin/stdout and doesn't serve as a web server.", MCPModeEnvKey)
	return nil
}

// PostProcess implements ParameterStore.
func (m *MCPParameters) PostProcess() error {
	if *m.MCPMode && *Job.JobMode {
		return errors.New("`--mcp-mode` and `--job-mode` cannot be used at the same time")
	}
	return nil
}

// MCPModeRequestedInRawArgs reports whether the MCP mode is requested, without parsing the flags.
// The global logger has to pick its destination before the parameter parsing step runs, because
// the parameter parsing initializer depends on the logger being ready, so it cannot read MCPMode.
func MCPModeRequestedInRawArgs() bool {
	return flag.HasRawCommandlineFlag(MCPModeFlagName) || flag.HasTruthyEnvironmentVariable(MCPModeEnvKey)
}

var _ ParameterStore = (*MCPParameters)(nil)

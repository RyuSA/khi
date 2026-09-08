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

package defaultinit

import (
	"context"
	"log/slog"
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/common/constants"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/mcp"
	"github.com/GoogleCloudPlatform/khi/pkg/mcp/khitools"
	mcpstdio "github.com/GoogleCloudPlatform/khi/pkg/mcp/stdio"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
)

// InitializerIDMCPRunner serves the Model Context Protocol over stdio.
const InitializerIDMCPRunner coreinit.InitializerID = "khi.default/mcp-runner"

// MCPRunnerInitializer starts the MCP stdio server when MCP mode is enabled.
var MCPRunnerInitializer = &coreinit.Initializer{
	ID: InitializerIDMCPRunner,
	Dependencies: []coreinit.InitializerID{
		InitializerIDInspectionTaskServer,
		InitializerIDParameterParse,
	},
	Init: func(ctx *coreinit.InitContext) error {
		mcpParams := coreinit.MustGet(ctx, MCPParametersKey)
		if !*mcpParams.MCPMode {
			return nil
		}
		// File form fields must resolve from local filesystem paths rather than upload tokens,
		// exactly as in job mode, because the command this server generates runs in job mode.
		upload.DefaultUploadFileStore = upload.NewJobModeStore()

		inspectionServer := coreinit.MustGet(ctx, InspectionTaskServerKey)
		tools, err := khitools.NewTools(khitools.Dependencies{
			InspectionServer:         inspectionServer,
			BinaryPath:               khitools.ResolveBinaryPath(),
			DefaultExportDestination: khitools.DefaultExportDestination,
		})
		if err != nil {
			return err
		}
		server, err := mcp.NewServer(
			mcp.Implementation{Name: "khi", Title: "Kubernetes History Inspector", Version: constants.VERSION},
			khitools.ServerInstructions,
			tools,
		)
		if err != nil {
			return err
		}

		ctx.OnRun(func(runCtx context.Context) error {
			// This message goes to stderr: the logger initializer redirects everything there when
			// MCP mode is requested, because stdout carries the JSON-RPC stream.
			slog.Info("Starting Kubernetes History Inspector as MCP stdio server...")
			return mcpstdio.Serve(runCtx, os.Stdin, os.Stdout, server)
		})
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(MCPRunnerInitializer)
}

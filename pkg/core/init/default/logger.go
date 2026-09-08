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
	"log/slog"
	"os"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

// InitializerIDLogger initializes the global logger.
const InitializerIDLogger coreinit.InitializerID = "khi.default/logger"

// LoggerInitializer initializes the global KHI logger.
var LoggerInitializer = &coreinit.Initializer{
	ID: InitializerIDLogger,
	Init: func(ctx *coreinit.InitContext) error {
		// This initializer runs before the command line arguments are parsed, because
		// ParameterParseInitializer depends on the logger being ready. The raw arguments
		// are inspected directly to decide the log destination.
		//
		// In MCP mode stdout carries the JSON-RPC stream, so every log line goes to stderr.
		if parameters.MCPModeRequestedInRawArgs() {
			logger.DefaultLogDestination = os.Stderr
		}
		logger.InitGlobalKHILogger()
		slog.Info("Initializing Kubernetes History Inspector...")
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(LoggerInitializer)
}

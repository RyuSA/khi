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

// Package khitools implements the MCP tools exposing KHI's inspection parameter schema.
//
// The tools are stateless: each call creates an inspection, uses it, and deletes it again.
// InspectionTaskServer keeps runners in a map with no eviction, so a session handle handed to a
// client would leak one runner per exploration step. The caller already holds the parameter values
// it is iterating on, so replaying them costs nothing and a dry run persists nothing.
package khitools

import (
	"log/slog"
	"os"
	"path/filepath"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
)

// DefaultExportDestination is the .khi path written into a generated command when the caller
// does not ask for a specific one.
const DefaultExportDestination = "output.khi"

// Dependencies carries what every tool in this package needs.
type Dependencies struct {
	// InspectionServer is the engine the tools read the inspection types, features and parameter
	// schema from.
	InspectionServer *coreinspection.InspectionTaskServer
	// BinaryPath is written at the head of every generated job mode command.
	BinaryPath string
	// DefaultExportDestination is the .khi path used when the caller does not specify one.
	DefaultExportDestination string
}

// ResolveBinaryPath returns the absolute path of the running KHI executable, to be written into
// the generated commands. It falls back to a relative "./khi" when the path cannot be determined,
// which still matches how the web UI presents the command.
func ResolveBinaryPath() string {
	executable, err := os.Executable()
	if err != nil {
		slog.Warn("failed to resolve the KHI executable path, falling back to ./khi", "error", err)
		return "./khi"
	}
	resolved, err := filepath.Abs(executable)
	if err != nil {
		slog.Warn("failed to make the KHI executable path absolute, falling back to ./khi", "error", err)
		return "./khi"
	}
	return resolved
}

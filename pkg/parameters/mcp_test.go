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

import "testing"

func TestMCPParametersPostProcess(t *testing.T) {
	testCases := []struct {
		name    string
		mcpMode bool
		jobMode bool
		wantErr bool
	}{
		{name: "neither mode", mcpMode: false, jobMode: false, wantErr: false},
		{name: "mcp mode alone", mcpMode: true, jobMode: false, wantErr: false},
		{name: "job mode alone", mcpMode: false, jobMode: true, wantErr: false},
		{name: "both modes are mutually exclusive", mcpMode: true, jobMode: true, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Job.JobMode is a package level pointer read by MCPParameters.PostProcess.
			previousJob := Job.JobMode
			t.Cleanup(func() { Job.JobMode = previousJob })
			jobMode := tc.jobMode
			Job.JobMode = &jobMode

			mcpMode := tc.mcpMode
			store := &MCPParameters{MCPMode: &mcpMode}

			err := store.PostProcess()
			if (err != nil) != tc.wantErr {
				t.Errorf("PostProcess() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

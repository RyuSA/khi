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
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

// setHeadlessModeParameters puts both mode parameter stores into the init context.
// Every server side initializer consults isHeadless, which reads both, so a test setting only
// one of them panics.
func setHeadlessModeParameters(ctx *coreinit.InitContext, jobMode bool, mcpMode bool) {
	coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{JobMode: &jobMode})
	coreinit.Set(ctx, MCPParametersKey, &parameters.MCPParameters{MCPMode: &mcpMode})
}

func TestIsHeadless(t *testing.T) {
	testCases := []struct {
		name    string
		jobMode bool
		mcpMode bool
		want    bool
	}{
		{name: "web server mode", jobMode: false, mcpMode: false, want: false},
		{name: "job mode", jobMode: true, mcpMode: false, want: true},
		{name: "mcp mode", jobMode: false, mcpMode: true, want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			ctx := engine.Context()
			setHeadlessModeParameters(ctx, tc.jobMode, tc.mcpMode)

			if got := isHeadless(ctx); got != tc.want {
				t.Errorf("isHeadless() = %v, want %v", got, tc.want)
			}
		})
	}
}

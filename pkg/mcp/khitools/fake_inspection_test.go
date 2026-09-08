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
	"context"
	"strings"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// The fake inspection type below stands in for the real ones in the tool tests.
//
// The real inspection types call Google Cloud APIs during a dry run, so they cannot be asserted
// on without credentials and a network. A fake type also keeps these tests from churning every
// time an unrelated production task changes its parameters.
const (
	fakeInspectionTypeID = "fake-inspection-type"
	fakeClusterFieldID   = "khi.example.com/fake/cluster-name"
	fakeKindsFieldID     = "khi.example.com/fake/kinds"

	// The label selector is how a task declares which inspection types it belongs to.
	fakeInspectionTypeLabelKey   = "khi.example.com/fake/environment"
	fakeInspectionTypeLabelValue = "fake"
)

// fakeFeatureID is the full implementation ID of the fake feature task, as FeatureList reports it.
const fakeFeatureID = "khi.example.com/fake/feature#default"

// newFakeInspectionServer builds an inspection server exposing only the fake inspection type.
func newFakeInspectionServer(t *testing.T) *coreinspection.InspectionTaskServer {
	t.Helper()
	// The dry run registers per task loggers, which panics unless the global logger exists.
	logger.InitGlobalKHILogger()
	server, err := coreinspection.NewServer(&inspectioncore_contract.IOConfig{
		ApplicationRoot: t.TempDir(),
		DataDestination: t.TempDir(),
		TemporaryFolder: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create the inspection server: %v", err)
	}
	if err := server.AddInspectionType(coreinspection.InspectionType{
		Id:          fakeInspectionTypeID,
		Name:        "Fake Inspection Type",
		Description: "An inspection type used by the KHI MCP tool tests.",
		Labels:      map[string]string{fakeInspectionTypeLabelKey: fakeInspectionTypeLabelValue},
	}); err != nil {
		t.Fatalf("failed to register the fake inspection type: %v", err)
	}

	forInspectionType := inspectioncore_contract.InspectionTypeLabelSelector(
		map[string]string{fakeInspectionTypeLabelKey: fakeInspectionTypeLabelValue})

	// A text parameter that is required, so an empty values object produces a blocking error.
	clusterNameTask := formtask.NewTextFormTaskBuilder(
		taskid.NewDefaultImplementationID[string](fakeClusterFieldID), 1000, "Cluster name").
		WithDescription("The name of the cluster to inspect.").
		WithValidator(func(ctx context.Context, value string) (string, error) {
			if strings.TrimSpace(value) == "" {
				return "Cluster name must not be empty", nil
			}
			return "", nil
		}).
		Build(forInspectionType)

	// A set parameter, so the wrong JSON type can be exercised.
	kindsTask := formtask.NewSetFormTaskBuilder(
		taskid.NewDefaultImplementationID[[]string](fakeKindsFieldID), 900, "Kinds").
		WithDescription("The resource kinds to gather.").
		WithOptionsSimple([]string{"pods", "deployments"}).
		WithDefaultValueConstant([]string{"pods"}, true).
		Build(forInspectionType)

	featureTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("khi.example.com/fake/feature"),
		[]taskid.UntypedTaskReference{
			taskid.NewTaskReference[string](fakeClusterFieldID),
			taskid.NewTaskReference[[]string](fakeKindsFieldID),
		},
		func(ctx context.Context) (any, error) { return nil, nil },
		forInspectionType,
		inspectioncore_contract.FeatureTaskLabel("Fake feature", "Gathers nothing at all.", 1, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)

	for _, task := range []coretask.UntypedTask{clusterNameTask, kindsTask, featureTask} {
		if err := server.AddTask(task); err != nil {
			t.Fatalf("failed to register a fake task: %v", err)
		}
	}
	return server
}

// newFakeDependencies builds the tool dependencies backed by the fake inspection server.
func newFakeDependencies(t *testing.T) Dependencies {
	t.Helper()
	return Dependencies{
		InspectionServer:         newFakeInspectionServer(t),
		BinaryPath:               "/usr/local/bin/khi",
		DefaultExportDestination: DefaultExportDestination,
	}
}

// assertHintTypeIsKnown guards the assumption that the hint type strings used in the assertions
// match the constants KHI actually emits.
func assertHintTypeIsKnown(t *testing.T, hintType string) {
	t.Helper()
	switch inspectionmetadata.ParameterHintType(hintType) {
	case inspectionmetadata.None, inspectionmetadata.Info, inspectionmetadata.Warning, inspectionmetadata.Error:
		return
	default:
		t.Errorf("unknown hint type %q", hintType)
	}
}

// fakeValuesWithCluster builds a values map satisfying the fake type's required parameter.
func fakeValuesWithCluster(name string) map[string]any {
	return map[string]any{fakeClusterFieldID: name}
}

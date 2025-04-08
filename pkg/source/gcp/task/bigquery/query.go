// Copyright 2025 Google LLC
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

package bigquery

import (
	"context"
	"fmt"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateBigQueryJobQuery(projectId string) string {
	return fmt.Sprintf(`
resource.labels.project_id="%s"
resource.type="bigquery_resource"
protoPayload.serviceData.jobCompletedEvent.eventName="query_job_completed"
protoPayload.serviceData.jobCompletedEvent.job.jobStatus.state="DONE"
`, projectId)
}

var BigQueryJobQueryTask = query.NewQueryGeneratorTask(BigQueryJobQueryTaskID, "BigQuery Query Job logs", enum.LogTypeBigQueryResource, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID, // need a project id form
}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	projectId := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
	return []string{GenerateBigQueryJobQuery(projectId)}, nil
}, GenerateBigQueryJobQuery("google-cloud-project-id"))

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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"gopkg.in/yaml.v3"
)

type bigqueryJobParser struct{}

// Dependencies implements parser.Parser.
func (b *bigqueryJobParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parser.Parser.
func (b *bigqueryJobParser) Description() string {
	return "Bigquery Jobs"
}

// GetParserName implements parser.Parser.
func (b *bigqueryJobParser) GetParserName() string {
	return "BigQuery Jobs Parser"
}

// Grouper implements parser.Parser.
func (b *bigqueryJobParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// LogTask implements parser.Parser.
func (b *bigqueryJobParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return BigQueryJobQueryTaskID.GetTaskReference()
}

// Parse implements parser.Parser.
func (b *bigqueryJobParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	// Parse BigQuery job from LogEntity
	job, err := NewBigQueryJobFromYamlStrings(l)
	if err != nil {
		return err
	}

	// Execution User
	requester := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")

	// BigQuery resource path
	resourcePath := job.ToResourcePath()
	body, _ := l.GetChildYamlOf("protoPayload.serviceData.jobCompletedEvent.job")

	// Record New Revision: Inserted BQ job
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobCreate,
		State:      enum.RevisionStateBigQueryJobPending,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.CreateTime),
		Partial:    false,
		Body:       body,
	})

	// Record New Revision: Started BQ job
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobStart,
		State:      enum.RevisionStateBigQueryJobRunning,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.StartTime),
		Partial:    false,
		Body:       body,
	})

	var state enum.RevisionState
	// status.error must be "" when the job succeeded
	// otherwise status.error must be an object
	isSuccess := job.Status.Error.Code == 0 && job.Status.Error.Message == ""
	if isSuccess {
		state = enum.RevisionStateBigQueryJobSuccess
	} else {
		state = enum.RevisionStateBigQueryJobFailed
	}

	// Record New Revision: Finished BQ job
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbBigQuryJobDone,
		State:      state,
		Requestor:  requester,
		ChangeTime: parseTime(job.Statistics.EndTime),
		Partial:    false,
		Body:       body,
	})

	var str string
	if isSuccess {
		str = "success"
	} else {
		str = "failed"
		cs.RecordLogSeverity(enum.SeverityError)
	}

	// This summary shows on right panel
	cs.RecordLogSummary(fmt.Sprintf("BigQuery Job(%s) finished with status %s", job.Name.JobId, str))

	// This event shows on the timeline view as ♢
	cs.RecordEvent(resourcePath)
	return nil
}

// TargetLogType implements parser.Parser.
func (b *bigqueryJobParser) TargetLogType() enum.LogType {
	return enum.LogTypeEvent
}

var _ parser.Parser = (*bigqueryJobParser)(nil)

func NewBigQueryJobFromYamlStrings(l *log.LogEntity) (*BigQueryJob, error) {
	jobYaml, err := l.GetChildYamlOf("protoPayload.serviceData.jobCompletedEvent.job")
	if err != nil {
		return nil, err
	}
	var job = &BigQueryJob{}
	err = yaml.Unmarshal([]byte(jobYaml), job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func parseTime(timeString string) time.Time {
	t, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		panic(err)
	}

	return t
}

var BigQueryJobParserTask = parser.NewParserTaskFromParser(BigQueryJobParserTaskID, &bigqueryJobParser{}, true, []string{InspectionTypeId})

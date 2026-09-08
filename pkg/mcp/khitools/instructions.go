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

// ServerInstructions is returned in the initialize result and is the only place this server
// explains itself. Every harness KHI targets folds it into the model context at handshake time,
// so an agent knows the loop without spending a tool call on it.
//
// Two properties of the text are load bearing, so keep them when editing:
//
//   - The loop must complete within the first InstructionsPrefixBudget characters. Codex treats
//     that prefix as the guidance it has available while deciding how to use the server, so a
//     loop sentence running past it is cut in half exactly where it matters most.
//     TestServerInstructionsLoopFitsInThePrefixBudget guards this.
//   - Every tool name must appear here. A tool the instructions never mention is invisible until
//     the model reads tools/list closely. TestServerInstructionsNameEveryTool guards this.
const ServerInstructions = `Kubernetes History Inspector (KHI) turns Kubernetes and Google Cloud logs into an interactive timeline of a cluster. This server does not run inspections: it returns the parameter schema and builds the "khi --job-mode" command line for you to run yourself.

Loop: khi_list_inspection_types -> khi_list_features -> khi_prepare_job_command with "values": {} to discover the parameters -> fill them in and repeat until "ready" is true -> run the returned "jobCommand".

Never guess parameter ids. They are fully qualified task reference ids such as "cloud.google.com/common/input-project-id", not short names like "projectId", and the valid set changes with the inspection type, the enabled features and live cloud state. Always take them from khi_prepare_job_command, and match every value to the "valueJSONType" it reports.

The time range is an end time plus a duration, not a start and end pair. The "gcp-*" inspection types read Google Cloud Logging and need Application Default Credentials. File parameters take a path on the machine that will run the generated command.`

// InstructionsPrefixBudget is the number of leading characters of ServerInstructions that a client
// is assumed to always have available. Codex documents this figure and asks servers to keep that
// prefix self-contained; the other harnesses read the whole string, so it is a floor rather than
// a limit.
const InstructionsPrefixBudget = 512

// instructionsLoopSentenceEnd is the tail of the sentence describing the tool loop. The test
// asserting the prefix budget locates the loop through it.
const instructionsLoopSentenceEnd = `run the returned "jobCommand".`

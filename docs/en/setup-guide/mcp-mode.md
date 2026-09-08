# MCP Mode Guide (For AI Agents)

KHI includes an **MCP mode** that exposes KHI's inspection parameter schema over the [Model Context Protocol](https://modelcontextprotocol.io/), so that an AI coding agent such as Claude Code, Codex or Antigravity can build a KHI [Job mode](./job-mode.md) command from an incident report.

## What MCP mode does, and what it does not

MCP mode **describes** KHI and **generates** the `khi --job-mode` command line. It does **not** run the inspection. The agent runs the generated command itself and then opens the resulting `.khi` file in the KHI Web UI.

This split exists because the Job mode parameter schema is the hard part. Parameter keys are fully qualified task reference IDs such as `cloud.google.com/common/input-project-id`, and the applicable set changes with the inspection type, the enabled features and live cloud state. A static description of it in an agent prompt goes stale as soon as the code changes, so KHI serves the schema itself.

## Starting the server

```bash
khi --mcp-mode
```

The server speaks JSON-RPC over stdin and stdout. All logging goes to stderr, because stdout carries the protocol stream.

`--mcp-mode` and `--job-mode` are mutually exclusive. No web server is started in MCP mode.

## Registering with an agent

Use the absolute path of the KHI binary; the agent starts KHI as a subprocess.

### Claude Code

```bash
claude mcp add khi -- /absolute/path/to/khi --mcp-mode
```

### Codex

`~/.codex/config.toml`:

```toml
[mcp_servers.khi]
command = "/absolute/path/to/khi"
args = ["--mcp-mode"]
```

### Antigravity / Gemini CLI

In the `mcpServers` object of the MCP settings JSON:

```json
{
  "mcpServers": {
    "khi": {
      "command": "/absolute/path/to/khi",
      "args": ["--mcp-mode"]
    }
  }
}
```

## Tools

| Tool | Purpose | Credentials |
| --- | --- | --- |
| `khi_list_inspection_types` | Lists the platforms KHI can gather logs from | Not required |
| `khi_list_features` | Lists the log sources an inspection type can collect | Not required |
| `khi_prepare_job_command` | Returns the parameter schema, validates the given values, and builds the command once nothing is left to fix | Required for `gcp-*` types |

There is no tool that explains the server. KHI returns that explanation as the MCP `instructions` field during the initialize handshake, and every harness listed above folds it into the model context automatically, so an agent knows the loop before it calls anything.

## The loop

1. `khi_list_inspection_types` picks the platform, for example `gcp-gke`.
2. `khi_list_features` picks the log sources. Each feature description says which logs it reads and what it helps investigate, so an agent can map them onto the incident. `["ALL"]` enables everything.
3. `khi_prepare_job_command` with `"values": {}` returns every parameter with its type, description, default and a hint explaining what is wrong with the current value.
4. The agent fills in `values` and calls again, until `"ready": true`. The `blockingErrors` array names exactly what still needs attention.
5. The agent runs the returned `jobCommand`, which writes the `.khi` file.

Validation results are reported as per-parameter hints rather than as a failure, so step 3 and step 4 are the same call and the agent can converge on a valid command without guessing.

## Requirements and caveats

- The `gcp-*` inspection types read Google Cloud Logging and require Application Default Credentials. Run `gcloud auth application-default login` before using them. `khi_prepare_job_command` performs a dry run that queries live cloud APIs for autocompletion and log volume estimation, so it needs network access and is not instantaneous.
- MCP mode is incompatible with the interactive OAuth flow (`--oauth-*`), because there is no browser to answer the login prompt. Use Application Default Credentials instead.
- File parameters take a local filesystem path on the machine that will run the generated command, exactly as in Job mode. Unlike the command shown in the Web UI, the command generated here keeps the real path the agent supplied instead of substituting a `path/to/file` placeholder.
- The tools are stateless. Every call passes the full parameter set, and nothing is retained between calls.

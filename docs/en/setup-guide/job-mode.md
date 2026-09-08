# Job Mode Guide (For CI/CD and Automation)

KHI includes a **Job mode** that performs log analysis and generates a `.khi` file directly at a specified path without starting the web server.

Job mode is useful for automated workflows, such as generating `.khi` files when alerts are triggered or capturing inspection snapshots during CI/CD pipeline runs (deployments, tests, etc.). The generated `.khi` file can later be uploaded to the KHI Web UI for interactive analysis.

## Obtaining Job Mode Commands

The parameter names accepted by `--job-inspection-values` are fully qualified task reference IDs, and which ones apply depends on the inspection type, the enabled features and live cloud state. Rather than writing them by hand, obtain the command from KHI itself in one of two ways:

- **Web UI**: fill out the parameters on the "New Inspection" page, and a Job mode CLI command representing those parameters is generated at the bottom of the form.
- **MCP mode**: run KHI as an MCP server so an AI agent can query the parameter schema and build the command. See [the MCP mode guide](./mcp-mode.md).

![Job Mode in KHI UI](../../images/job-mode.png)

> [!NOTE]
> The CLI command displayed in the UI uses a direct binary execution format (e.g., `./khi ...`). When running via Docker, mount the output directory as shown below.

## Running in Docker Containers

Mount the output directory into the container (e.g., `-v $(pwd):/output`):

```bash
docker run --rm \
  -v $(pwd):/output \
  gcr.io/kubernetes-history-inspector/release:latest \
  --job-mode \
  --job-inspection-type="gcp-gke" \
  --job-inspection-features="ALL" \
  --job-inspection-values='{
    "cloud.google.com/common/input-project-id": "my-gcp-project",
    "cloud.google.com/k8s/input-cluster-name": "my-cluster",
    "cloud.google.com/common/input-end-time": "2026-09-01T12:00:00Z",
    "cloud.google.com/common/input-duration": "3h"
  }' \
  --job-export-destination="/output/result.khi"
```

> [!NOTE]
> The values above are an illustration, not a complete list. Parameter keys are task reference IDs such as `cloud.google.com/common/input-project-id`, never short names like `projectId`, and the applicable set changes with the inspection type and the enabled features. Always take the authoritative set from the Web UI or from MCP mode.
>
> The time range is expressed as an end time plus a duration rather than a start and end pair: `cloud.google.com/common/input-end-time` takes an RFC3339 timestamp and `cloud.google.com/common/input-duration` takes a Go duration such as `3h` or `90m`.

> [!IMPORTANT]
> **Replacing File Placeholders and Mounting Input Files**
>
> When the inspection parameters include local files (such as uploaded log files), the generated command contains `"path/to/file"` as a placeholder in `--job-inspection-values`.
> When executing via Docker, replace `"path/to/file"` with the actual input file path mounted inside the container:
>
> ```bash
> docker run --rm \
>   -v $(pwd):/output \
>   -v /path/to/local/audit.log:/input/audit.log:ro \
>   gcr.io/kubernetes-history-inspector/release:latest \
>   --job-mode \
>   --job-inspection-type="oss-kubernetes-from-files" \
>   --job-inspection-features="ALL" \
>   --job-inspection-values='{"khi.google.com/oss/form/kube-apiserver-audit-log-files":"/input/audit.log"}' \
>   --job-export-destination="/output/result.khi"
> ```

## Inspection Types

`--job-inspection-type` takes one of the following IDs:

| ID | Description |
| --- | --- |
| `gcp-gke` | Google Kubernetes Engine |
| `gcp-composer` | Cloud Composer (Managed Airflow) |
| `gcp-gke-on-aws` | GKE on AWS (Anthos on AWS) |
| `gcp-gke-on-azure` | GKE on Azure (Anthos on Azure) |
| `gcp-gdcv-for-baremetal` | GDCV for Baremetal |
| `gcp-gdcv-for-vmware` | GDCV for VMware |
| `oss-kubernetes-from-files` | OSS Kubernetes log files |

## Parameter Details

`--job-inspection-features` takes a comma separated list of feature task IDs, or the single value `ALL` to enable every feature available for the inspection type. Feature IDs include the implementation suffix after `#`, for example `cloud.google.com/log/k8s-node/tail#default`.

For the definitions and specifications of all command line flags used in Job mode, see [pkg/parameters/job.go](../../../pkg/parameters/job.go).

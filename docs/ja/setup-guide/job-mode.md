# Job モードガイド（CI/CD・自動化向け）

KHI には、Web サーバを起動せずに指定したログの分析・可視化処理を行い、直接 `.khi` ファイルを出力する **Job モード** が用意されています。

Job モードを使用すると、アラート通知時や CI/CD パイプライン（デプロイ時・テスト時等）のトリガーに合わせて自動的に `.khi` ファイルを生成・保存できます。生成された `.khi` ファイルは、後から KHI Web UI にアップロードしてインタラクティブに分析できます。

## Job モードのコマンドを取得する方法

`--job-inspection-values` が受け付けるパラメータ名は完全修飾のタスク参照 ID であり、どの ID が有効かはインスペクションタイプ・有効化した機能・クラウド側の状態によって変化します。手書きせず、KHI 自身から以下のいずれかの方法で取得してください。

- **Web UI**: 「新規インスペクション作成」画面でパラメータを入力すると、画面下部に同等の設定で実行できる Job モードの CLI コマンドが表示されます。
- **MCP モード**: KHI を MCP サーバーとして起動し、AI エージェントにパラメータスキーマを問い合わせさせてコマンドを組み立てます。[MCP モードガイド](./mcp-mode.md)を参照してください。

![Job Mode in KHI UI](../../images/job-mode.png)

> [!NOTE]
> UI 上に表示されるコマンド例ではバイナリ直実行（`./khi ...` 等）の形式となっています。Docker コンテナから実行する場合は、出力先ディレクトリをマウントして実行してください。

## Docker コンテナでの実行例

出力先ディレクトリをコンテナにマウント（例: `-v $(pwd):/output`）して実行します。

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
> 上記の値は例示であり、網羅的な一覧ではありません。パラメータのキーは `cloud.google.com/common/input-project-id` のようなタスク参照 ID であり、`projectId` のような短縮名ではありません。また有効なキーの集合はインスペクションタイプや有効化した機能によって変化します。正確な一覧は必ず Web UI または MCP モードから取得してください。
>
> 時間範囲は開始時刻と終了時刻の組ではなく、終了時刻と期間で指定します。`cloud.google.com/common/input-end-time` は RFC3339 形式のタイムスタンプ、`cloud.google.com/common/input-duration` は `3h` や `90m` のような Go の duration 文字列です。

> [!IMPORTANT]
> **入力ファイルパスの置き換えとマウントについて**
>
> ログファイルのアップロード等、パラメータにローカルファイルが含まれる場合、生成されるコマンドの `--job-inspection-values` 内には `"path/to/file"` というプレースホルダーが出力されます。
> Docker コンテナで実行する際は、この `"path/to/file"` を実際の入力ファイルのパス（コンテナ内にマウントしたパス）へ置き換える必要があります。
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

## インスペクションタイプ一覧

`--job-inspection-type` には以下の ID を指定します。

| ID | 説明 |
| --- | --- |
| `gcp-gke` | Google Kubernetes Engine |
| `gcp-composer` | Cloud Composer (Managed Airflow) |
| `gcp-gke-on-aws` | GKE on AWS (Anthos on AWS) |
| `gcp-gke-on-azure` | GKE on Azure (Anthos on Azure) |
| `gcp-gdcv-for-baremetal` | GDCV for Baremetal |
| `gcp-gdcv-for-vmware` | GDCV for VMware |
| `oss-kubernetes-from-files` | OSS Kubernetes ログファイル |

## パラメータ詳細

`--job-inspection-features` にはフィーチャータスク ID をカンマ区切りで指定するか、インスペクションタイプで利用可能なすべての機能を有効にする `ALL` を指定します。フィーチャー ID には `cloud.google.com/log/k8s-node/tail#default` のように `#` 以降の実装サフィックスまで含める必要があります。

Job モードで使用可能な各コマンドラインフラグの定義については [pkg/parameters/job.go](../../../pkg/parameters/job.go) を参照してください。

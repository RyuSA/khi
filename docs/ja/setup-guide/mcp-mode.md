# MCP モードガイド（AI エージェント向け）

KHI には、インスペクションのパラメータスキーマを [Model Context Protocol](https://modelcontextprotocol.io/) 経由で公開する **MCP モード** が用意されています。Claude Code・Codex・Antigravity のような AI コーディングエージェントが、インシデント情報から KHI の [Job モード](./job-mode.md)コマンドを組み立てられるようになります。

## MCP モードがすること・しないこと

MCP モードは KHI の使い方を**説明**し、`khi --job-mode` のコマンドラインを**生成**します。インスペクションの**実行はしません**。生成されたコマンドはエージェント自身が実行し、出力された `.khi` ファイルを KHI Web UI で開きます。

この切り分けは、Job モードで難しいのがパラメータスキーマだからです。パラメータのキーは `cloud.google.com/common/input-project-id` のような完全修飾のタスク参照 ID であり、有効なキーの集合はインスペクションタイプ・有効化した機能・クラウド側の状態によって変化します。これをエージェントのプロンプトに静的に書くと実装の変更ですぐ陳腐化するため、KHI 自身にスキーマを喋らせています。

## サーバーの起動

```bash
khi --mcp-mode
```

サーバーは stdin/stdout 上で JSON-RPC を話します。stdout はプロトコルのストリームに占有されるため、ログはすべて stderr に出力されます。

`--mcp-mode` と `--job-mode` は同時に指定できません。MCP モードでは Web サーバーは起動しません。

## エージェントへの登録

エージェントは KHI を子プロセスとして起動するため、KHI バイナリの絶対パスを指定してください。

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

MCP 設定 JSON の `mcpServers` に記述します。

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

## ツール一覧

| ツール | 役割 | 認証情報 |
| --- | --- | --- |
| `khi_list_inspection_types` | KHI がログを収集できるプラットフォームの一覧を返す | 不要 |
| `khi_list_features` | インスペクションタイプが収集できるログソースの一覧を返す | 不要 |
| `khi_prepare_job_command` | パラメータスキーマを返し、与えられた値を検証し、問題がなくなればコマンドを生成する | `gcp-*` タイプでは必要 |

サーバー自身の使い方を説明するツールはありません。その説明は initialize ハンドシェイク時に MCP の `instructions` フィールドとして返され、上記のいずれのハーネスも自動的にモデルのコンテキストへ取り込むため、エージェントは最初のツール呼び出しの前からループを把握しています。

## 利用ループ

1. `khi_list_inspection_types` でプラットフォーム（例: `gcp-gke`）を選ぶ。
2. `khi_list_features` でログソースを選ぶ。各機能の説明にはどのログを読み、何の調査に役立つかが書かれているため、エージェントはインシデントの内容と対応付けられます。`["ALL"]` ですべて有効になります。
3. `khi_prepare_job_command` を `"values": {}` で呼ぶと、全パラメータが型・説明・デフォルト値、および現在の値の何が問題かを示す hint 付きで返ります。
4. エージェントが `values` を埋めて再度呼び出し、`"ready": true` になるまで繰り返します。`blockingErrors` 配列に、まだ対応が必要なパラメータが正確に列挙されます。
5. 返された `jobCommand` をエージェントが実行すると `.khi` ファイルが出力されます。

検証結果はエラーではなくパラメータごとの hint として返るため、手順 3 と手順 4 は同じ呼び出しになります。エージェントは推測せずに正しいコマンドへ収束できます。

## 前提条件と注意点

- `gcp-*` のインスペクションタイプは Google Cloud Logging を読むため、アプリケーションのデフォルト認証情報（ADC）が必要です。利用前に `gcloud auth application-default login` を実行してください。`khi_prepare_job_command` はオートコンプリートやログ件数の見積もりのために実際のクラウド API を叩く dry run を行うため、ネットワークが必要で、即座には完了しません。
- MCP モードは対話的な OAuth フロー（`--oauth-*`）と併用できません。ログインプロンプトに応答するブラウザが存在しないためです。ADC を利用してください。
- ファイルパラメータは Job モードと同様、生成されたコマンドを実行するマシン上のローカルファイルパスを受け取ります。Web UI が表示するコマンドとは異なり、ここで生成されるコマンドは `path/to/file` のプレースホルダーに置き換えず、エージェントが指定した実際のパスをそのまま保持します。
- ツールはステートレスです。呼び出しごとにパラメータ一式を渡す必要があり、呼び出し間で状態は保持されません。

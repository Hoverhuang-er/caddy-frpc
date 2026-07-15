# caddy-frpc

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![Caddy](https://img.shields.io/badge/Caddy-v2.11.4-purple?logo=caddy)](https://caddyserver.com)
[![frp](https://img.shields.io/badge/frp-dev-orange)](https://github.com/fatedier/frp)
[![License](https://img.shields.io/badge/License-Apache%202.0-green)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hxgm/caddy-frpc.svg)](https://pkg.go.dev/github.com/hxgm/caddy-frpc)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)](https://github.com/hxgm/caddy-frpc/pulls)
[![CI](https://github.com/Hoverhuang-er/caddy-frpc/actions/workflows/ci.yml/badge.svg)](https://github.com/Hoverhuang-er/caddy-frpc/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-blue?logo=docker)](https://github.com/Hoverhuang-er/caddy-frpc/pkgs/container/caddy-frpc)
[![AI Native](https://img.shields.io/badge/AI%20Native-000000?logo=data:image/svg%2bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBkPSJNOCAxYy4zOSAwIC43NS4yMy45LjU4bDEuNDcgMy41MiAzLjUzIDEuNDdjLjM1LjE1LjU4LjUxLjU4LjkgMCAuMzktLjIzLjc1LS41OC45bC0zLjUzIDEuNDctMS40NyAzLjUyYy0uMTUuMzUtLjUxLjU4LS45LjU4cy0uNzUtLjIzLS45LS41OGwtMS40Ny0zLjUyLTMuNTMtMS40N2MtLjM1LS4xNS0uNTgtLjUxLS41OC0uOXMuMjMtLjc1LjU4LS45bDMuNTMtMS40N0w3LjEgMS41OGMuMTUtLjM1LjUxLS41OC45LS41OHoiLz48L3N2Zz4=)](https://github.com/Hoverhuang-er/caddy-frpc)

[English](README.md) | [中文版](README_zh.md)


## クイックスタート

### 方法 1: xcaddy でビルド（推奨）

[xcaddy](https://github.com/caddyserver/xcaddy) を使用して、caddy-frpc モジュールを組み込んだ Caddy バイナリをソースからビルドします。Caddy のバージョンと caddy-frpc のバージョンを完全に制御できます。最新のコードが必要な場合や開発時に使用します。

```bash
# 最新リリース
xcaddy build v2.11.4 --with github.com/Hoverhuang-er/caddy-frpc@latest

# 特定のバージョンを指定
xcaddy build v2.11.4 --with github.com/Hoverhuang-er/caddy-frpc@v0.1.2

# ローカル開発版
xcaddy build v2.11.4 --with github.com/Hoverhuang-er/caddy-frpc@.
```

プラグインの埋め込みを確認：
```bash
./caddy list-modules | grep frpc
# 出力: caddy.apps.frpc
```

### 方法 2: プリビルドバイナリをダウンロード

[GitHub Releases](https://github.com/Hoverhuang-er/caddy-frpc/releases) からプリビルドされたバイナリをダウンロード。ビルドツールは不要です。Linux、macOS、Windows の AMD64 および ARM64 に対応。

```bash
# Linux AMD64
wget -O caddy https://github.com/Hoverhuang-er/caddy-frpc/releases/latest/download/caddy_wh_frpc_linux_amd64
chmod +x caddy

# macOS ARM64 (Apple Silicon)
wget -O caddy https://github.com/Hoverhuang-er/caddy-frpc/releases/latest/download/caddy_wh_frpc_darwin_arm64
chmod +x caddy
```

確認：
```bash
./caddy list-modules | grep frpc
# 出力: caddy.apps.frpc
```

### 方法 3: Docker

```bash
docker pull ghcr.io/hoverhuang-er/caddy-frpc:latest

docker run -v ./frpc.toml:/etc/caddy/frpc.toml \
  -v ./Caddyfile:/etc/caddy/Caddyfile \
  ghcr.io/hoverhuang-er/caddy-frpc:latest
```

### 方法 4: Kubernetes

ConfigMap を作成して frpc と Caddy の設定を保存し、デプロイ：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: caddy-config
data:
  Caddyfile: |
    { frpc ./frpc.toml }
    :8080 { reverse_proxy 127.0.0.1:8000 }
  frpc.toml: |
    serverAddr = "frps.example.com"
    serverPort = 7000
    auth.token = "my-token"
    [[visitors]]
    name = "my-service"
    type = "stcp"
    serverName = "remote-service"
    secretKey = "my-secret"
    bindAddr = "127.0.0.1"
    bindPort = 8000
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: caddy-frpc
spec:
  replicas: 1
  selector:
    matchLabels: { app: caddy-frpc }
  template:
    metadata:
      labels: { app: caddy-frpc }
    spec:
      containers:
        - name: caddy
          image: ghcr.io/hoverhuang-er/caddy-frpc:latest
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: config
              mountPath: /etc/caddy
      volumes:
        - name: config
          configMap:
            name: caddy-config
```

[English](README.md) | [中文版](README_zh.md)

[frpc](https://github.com/fatedier/frp) を Go ライブラリとして埋め込み、visitor モードで動作させる Caddy アプリケーションモジュール。**AI Native** — 設計の 100% がローカル Mem0 操作メモリからの要件発掘と Skill 駆動によって実現されました。

frpc visitor はローカル TCP リスナーを作成し、frps を介してリモート frpc クライアントに登録されたサービスへのトンネル接続を確立します。Caddy が frpc のライフサイクルを管理し、visitor リスナーへのリバースプロキシを通じて完全なミドルウェアチェーンを適用できます。

## アーキテクチャ

```mermaid
flowchart TB
    subgraph External["外部ネットワーク"]
        FRPS["frps (公開サーバー)"]
        FRPC_A["リモート frpc A<br/>proxy 登録 (stcp/xtcp/sudp)"]
    end

    subgraph CaddyProcess["Caddy プロセス"]
        subgraph CaddyHTTP["Caddy HTTP サーバー"]
            HTTP["HTTP リスナー (:8080)"]
            MW["ミドルウェアチェーン<br/>auth · headers · rewrite · proxy"]
            RP["reverse_proxy"]
        end

        subgraph CaddyFRPC["caddy.apps.frpc モジュール"]
            PROV["Provision()<br/>設定をロード"]
            START["Start()<br/>client.NewService"]
            VISITOR["Visitor Goroutine<br/>[visitor ごとに 1 つ]"]
            LISTENER["TCP リスナー<br/>bindAddr:bindPort"]
        end

        subgraph Config["設定"]
            CFG["frpc.toml / .yaml / .json / .ini"]
        end

        CFG -->|"loadConfig()"| PROV
        PROV -->|"frpcConfig"| START
        START -->|"source.Aggregator"| VISITOR
        VISITOR -->|"作成"| LISTENER
    end

    FRPS <==>|"frp 制御 + トンネル"| VISITOR
    FRPC_A <==>|"stcp/xtcp トンネル"| FRPS
    HTTP --> MW --> RP -->|"127.0.0.1:bindPort"| LISTENER
```

**データフロー：**
- Visitor は起動時に frps との制御接続を確立
- リモート frpc A が一致する proxy を登録すると、visitor はローカル `bindAddr:bindPort` に TCP リスナーを作成
- Caddy HTTP サーバーが visitor のローカルリスナーにリバースプロキシ
- 各 visitor は独立した goroutine で動作；すべてのチャネルは goroutine-safe

### 並行安全性モデル

```mermaid
flowchart LR
    subgraph Safe["並行安全性の保証"]
        MUTEX["sync.Mutex<br/>Start/Stop を保護"]
        CHAN["バッファ付き channel<br/>goroutine-safe"]
        CTX["context.Context<br/>キャンセル伝播"]
    end

    MUTEX --> START_STOP["非再入可能 Start/Stop"]
    CHAN --> VISITOR_IO["visitor I/O 非ブロッキング"]
    CTX --> GRACEFUL["Stop() で graceful shutdown"]
```

- `sync.Mutex` が共有状態 (`svr`, `cancel`) を `Start()`/`Stop()` の同時呼び出しから保護
- 各 visitor のチャネルは独立して接続をバッファリング—ホットパスに共有書き込みロックなし
- `context.Context` のキャンセルが Caddy のライフサイクルから全 visitor goroutine に伝播

## サポートする設定形式

モジュールは以下の形式の frpc 設定ファイルを受け入れます：

| 形式 | 拡張子 | 備考 |
|------|--------|------|
| TOML | `.toml` | frp v1 ネイティブ形式（推奨） |
| YAML | `.yaml` / `.yml` | |
| JSON | `.json` | |
| INI | `.ini` | レガシー形式、frp により非推奨 |

## Visitor モード

本モジュールは frpc **visitor** モード（STCP/XTCP/SUDP visitor）で動作します。

1. **frpc A** が frps にプロキシを登録します（type = stcp、secretKey 付き）
2. **Caddy-frpc**（本モジュール）が visitor を設定し、frps 経由で frpc A のサービスに接続します
3. visitor が `bindAddr:bindPort` にローカル TCP リスナーを作成します
4. Caddy の HTTP サーバーがそのローカルポートにリバースプロキシします

本モジュールは frpc プロキシモード（frpc が frps から作業接続を受信する方式）をサポートしません。設定ファイル内の `[[visitors]]` のみが処理され、`[[proxies]]` は警告を記録してスキップされます。

## 使い方

### 0. ビルド

```bash
xcaddy build v2.11.4 --with github.com/hxgm/caddy-frpc
```

frpc モジュールが組み込まれた `caddy` バイナリが生成されます。

### 1. frpc 設定ファイルを作成

標準的な frpc 設定ファイルを作成します。`[[visitors]]` のみが処理され、`[[proxies]]` は無視されます。

**TOML（推奨）：**

```toml
# frpc.toml
serverAddr = "frps.example.com"
serverPort = 7000
auth.token = "my-token"

[[visitors]]
name = "my-service"
type = "stcp"
serverName = "remote-service"
secretKey = "my-secret"
bindAddr = "127.0.0.1"
bindPort = 8000
```

**INI（レガシー形式）：**

```ini
; frpc.ini
[common]
server_addr = frps.example.com
server_port = 7000
token = my-token

[my-service]
type = stcp
role = visitor
server_name = remote-service
sk = my-secret
bind_addr = 127.0.0.1
bind_port = 8000
```

**YAML と JSON** 形式もサポートしています。詳細は `examples/` ディレクトリを参照してください。

### 2. Caddy を起動

起動方法は **3 通り**あります。環境に合わせて選択してください。

#### 方法 A：Caddyfile（推奨）

Caddyfile が frpc 設定ファイルを参照します。ここでは Caddyfile が必須です。

```caddyfile
# Caddyfile
{
    # Caddy に frpc モジュールを読み込ませ、frpc.toml を指定
    frpc ./frpc.toml
}

# 8080 番ポートの HTTP サーバーが visitor トンネルにプロキシ
:8080 {
    reverse_proxy 127.0.0.1:8000
}
```

```bash
./caddy run --config Caddyfile
```

#### 方法 B：Caddy JSON（frpc 設定をインラインで記述）

frpc 設定を Caddy の JSON 設定に直接埋め込みます。別途 frpc.toml は不要です。

```json
{
  "apps": {
    "frpc": {
      "config": "serverAddr = \"frps.example.com\"\nserverPort = 7000\n\n[[visitors]]\nname = \"my-service\"\ntype = \"stcp\"\nserverName = \"remote-service\"\nsecretKey = \"my-secret\"\nbindAddr = \"127.0.0.1\"\nbindPort = 8000"
    },
    "http": {
      "servers": {
        "srv0": {
          "listen": [":8080"],
          "routes": [
            {
              "handle": [{
                "handler": "reverse_proxy",
                "upstreams": [{"dial": "127.0.0.1:8000"}]
              }]
            }
          ]
        }
      }
    }
  }
}
```

```bash
./caddy run --config caddy.json
```

#### 方法 C：Caddyfile ブロック構文

```caddyfile
# Caddyfile
{
    frpc {
        config ./frpc.toml
    }
}

:8080 {
    reverse_proxy 127.0.0.1:8000
}
```

#### 重要なポイント：`--config` は常に Caddy 設定（Caddyfile または JSON）を指定します。frpc.toml ではありません。

| こう思うかもしれません | 実際の動作 |
|------------------------|-----------|
| `./caddy run --config frpc.toml` | Caddy が frpc.toml を Caddyfile として解析しようとし、失敗します |
| `./caddy run --config Caddyfile` | 正しい使い方。Caddy が Caddyfile を読み込み、Caddyfile が frpc.toml を参照します |


## 前提条件

- 稼働中の [frps](https://github.com/fatedier/frp) サーバー
- stcp/xtcp/sudp プロキシを登録したリモート frpc クライアントが少なくとも 1 つ
- [xcaddy](https://github.com/caddyserver/xcaddy)（ビルド用）
- Go 1.26+

## 設定リファレンス

完全な設定オプションは [frp ドキュメント](https://github.com/fatedier/frp#readme) を参照してください。主な visitor フィールド：

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | Visitor 名 |
| `type` | string | `stcp`、`xtcp`、または `sudp` |
| `serverName` | string | frps 上のターゲットプロキシ名 |
| `secretKey` | string | ターゲットプロキシと一致する共有秘密鍵 |
| `bindAddr` | string | ローカルバインドアドレス（デフォルト 127.0.0.1） |
| `bindPort` | int | ローカルバインドポート |

トップレベル設定の共通フィールド：

| フィールド | 型 | デフォルト | 説明 |
|-----------|-----|-----------|------|
| `serverAddr` | string | `0.0.0.0` | frps サーバーアドレス |
| `serverPort` | int | `7000` | frps サーバーポート |
| `auth.token` | string | | 認証トークン |
| `transport.protocol` | string | `tcp` | `tcp`、`kcp`、`quic`、`websocket` |

## テスト

サンプル設定はモジュールの設定ローダーでテストされています。実行：

```bash
go test -v -count=1 ./...
```

テストスイートは以下を検証します：
- TOML 設定の解析（examples/frpc.toml）
- INI 設定の解析（examples/frpc.ini）
- Caddyfile の解析（examples/Caddyfile）
- モジュール登録とインターフェース準拠
- Caddy JSON `config` フィールドによるインライン設定
- YAML および JSON 設定形式のサポート

## サンプル

完全なテスト済みサンプル設定は `examples/` ディレクトリを参照してください。

## ライセンス

Apache 2.0

# Caddy-frpc: 嵌入式 frpc 的 Caddy Listener Wrapper 插件

**日期:** 2026-07-15
**状态:** 设计阶段
**参考项目:** [mohammed90/caddy-ngrok-listener](https://github.com/mohammed90/caddy-ngrok-listener)

## 1. 概述

通过 xcaddy 插件机制，将 frpc (fatedier/frp 客户端) 的 Layer 7 HTTP/HTTPS 代理能力嵌入到 Caddy 中。插件以 `listener_wrapper` 模式工作，导入 frp 作为 Go 库，接收来自 frps 的工作连接并将其转化为 Caddy HTTP 服务器的 `net.Listener`，使得 Caddy 的完整中间件链可以处理通过 frp 隧道进入的请求。

## 2. 目标

- 将 frpc 作为嵌入式库集成到 Caddy，不启动独立的 frpc 进程
- 支持通过 `--config` 参数注入 `frpc.toml` / `frpc.ini` 配置文件
- 暴露 frpc HTTP/HTTPS 代理能力给 Caddy 的中间件链
- 遵循 `caddy-ngrok-listener` 的 `listener_wrapper` 模式
- 使 Caddy 可以处理：认证、Header 操作、模板、压缩、路由、反向代理

## 3. 非目标

- TCP/UDP/STCP/XTCP 代理类型支持（仅限于 Layer 7 HTTP/HTTPS）
- frps 服务端实现
- 热重载 frpc 配置（后续版本可扩展）
- frpc Admin API / 管理界面集成
- 多 frpc 实例管理
- frp V2 协议支持

## 4. 架构

### 4.1 模块结构

```
caddy.listeners.frpc                  ← 主模块 (ListenerWrapper)
  └─ tunnels.http                     ← HTTP 代理隧道子模块
  └─ tunnels.https                    ← HTTPS 代理隧道子模块
```

### 4.2 模块生命周期

```
Provision(ctx):
  ├─ 解析 --config 路径或 Caddyfile 中的配置路径
  ├─ 检测文件格式 (TOML/INI)，使用 frp 的 source.NewFileSource() 加载
  ├─ 解析出 ClientCommonConfig (frps 地址、token、传输配置等)
  ├─ 过滤出 type=http/https 的 ProxyConfig
  ├─ 创建对应数量的 frpcListener 适配器
  └─ 清理不需要的配置

WrapListener(existing net.Listener):
  ├─ 创建 frpc ServiceOptions
  ├─ 设置 ConnectorCreator (TCP/QUIC/KCP/WebSocket)
  ├─ 设置 HandleWorkConnCb → 路由到对应 listener 的 channel
  ├─ goroutine: client.NewService(options).Run(ctx)
  └─ 返回 frpcListener 代理给 Caddy
```

### 4.3 核心适配层: workConn → net.Listener

frpc 原生使用 `Proxy.InWorkConn(conn, msg)` 接收工作连接。frpc 已经提供了 `SetInWorkConnCallback` 钩子，当回调返回 `false` 时，连接不再转发到本地服务。

```go
// frpc 内部机制 (proxy/proxy.go)
func (pxy *BaseProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
    if pxy.inWorkConnCallback != nil {
        if !pxy.inWorkConnCallback(pxy.baseCfg, conn, m) {
            return  // 回调消费了连接
        }
    }
    pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey) // 转发到本地服务
}
```

适配器实现：

```go
type frpcListener struct {
    ch     chan net.Conn
    addr   net.Addr
    closed atomic.Bool
}

func (l *frpcListener) Accept() (net.Conn, error) {
    conn, ok := <-l.ch
    if !ok { return nil, net.ErrClosed }
    return conn, nil
}

func (l *frpcListener) Close() error {
    l.closed.Store(true)
    close(l.ch)
    return nil
}

func (l *frpcListener) Addr() net.Addr { return l.addr }
```

回调注入：

```go
svrOptions.HandleWorkConnCb = func(baseCfg *v1.ProxyBaseConfig, conn net.Conn, msg *msg.StartWorkConn) bool {
    listener, ok := listeners[baseCfg.Name]
    if !ok { return true } // 不是 HTTP 代理，按默认处理
    // 注入 frps 传递的原始客户端地址信息到连接上下文
    if msg.SrcAddr != "" && msg.SrcPort != 0 {
        conn = wrapWithRemoteAddr(conn, msg.SrcAddr, msg.SrcPort)
    }
    listener.ch <- conn
    return false // 消费了连接
}
```

### 4.4 HTTP 请求流

```
外部用户
  │  https://web.example.com
  ▼
frps (公网服务器)
  │  域名路由 (customDomains)
  │  URL 路径分派 (locations)
  │  TLS 终结 (HTTPS proxy)
  │
  ├─ work connection (frp 隧道)
  ▼
frpc Service (嵌入在 Caddy 进程中)
  │
  ▼
frpcListener.Accept() ← channel 拿到 net.Conn
  │
  ▼
Caddy HTTP Server
  ├─ 路由匹配
  ├─ 中间件链
  │  ├─ 认证 (basic auth, JWT, OIDC)
  │  ├─ Header 操作 (request/response)
  │  ├─ 模版
  │  ├─ 压缩
  │  ├─ 重写
  │  └─ 反向代理 / 静态文件
  └─ 响应回 frps → 外部用户
```

### 4.5 请求上下文增强

frps 在 `StartWorkConn` 消息中传递 `SrcAddr`/`SrcPort`（原始客户端地址）和 `DstAddr`/`DstPort`（目标地址）。

适配器需要：
1. 用 `net.TCPConn.SetRemoteAddr()` 或包装连接来设置正确的远程地址
2. Caddy 的 `{remote_host}` / `{remote_port}` 占位符可用

## 5. 配置

### 5.1 独立配置文件模式 (TOML)

用户运行：
```
caddy run --config ./frpc.toml
```

`frpc.toml` 是标准格式：

```toml
serverAddr = "frps.example.com"
serverPort = 7000
auth.token = "my-token"

[[proxies]]
name = "web"
type = "http"
localIP = "127.0.0.1"
localPort = 8080
customDomains = ["web.example.com"]
locations = ["/"]
hostHeaderRewrite = "internal.local"
```

插件通过文件扩展名自动检测格式 (`.toml` / `.ini`)。

### 5.2 Caddyfile 模式

```caddyfile
{
    servers :8080 {
        listener_wrappers {
            frpc ./frpc.toml
        }
    }
}

:8080 {
    root * /var/www
    file_server
}
```

### 5.3 Caddy JSON 模式

```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [":8080"],
          "listener_wrappers": [
            {
              "wrapper": "frpc",
              "config_file": "./frpc.toml"
            }
          ]
        }
      }
    }
  }
}
```

## 6. 配置解析

### 6.1 支持的配置字段

从 `ClientCommonConfig` 提取的必要字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `serverAddr` | string | frps 服务器地址 |
| `serverPort` | int | frps 端口 (默认 7000) |
| `auth.token` | string | 认证 token |
| `transport.protocol` | string | tcp/kcp/quic/websocket |
| `transport.tls.enable` | bool | TLS 加密 |
| `transport.poolCount` | int | 连接池大小 |

每个 HTTP/HTTPS 代理支持的 frp 配置：

| 字段 | 类型 | HTTP | HTTPS | 说明 |
|------|------|------|-------|------|
| `name` | string | ✓ | ✓ | 代理名称 |
| `customDomains` | []string | ✓ | ✓ | 域名路由 |
| `subdomain` | string | ✓ | ✓ | 子域名 |
| `locations` | []string | ✓ | ✗ | URL 路径 |
| `hostHeaderRewrite` | string | ✓ | ✗ | Host 重写 |
| `httpUser`/`httpPassword` | string | ✓ | ✗ | Basic Auth |
| `requestHeaders` | HeaderOp | ✓ | ✗ | 请求头操作 |
| `responseHeaders` | HeaderOp | ✓ | ✗ | 响应头操作 |

### 6.2 忽略的配置

- TCP/UDP/STCP/XTCP 代理 → warn 并跳过
- `localIP`/`localPort` → 被 Caddy 中间件取代（request 不进本地服务）
- frpc 的 `webServer` (admin UI) → 不创建
- Visitor 类型 → 不处理

## 7. 错误处理与生命周期

| 场景 | 行为 |
|------|------|
| frps 不可达 | frpc 库内置指数退避重试 |
| Caddy 优雅关闭 | frpc Service.Stop() 通过 context cancellation 触发 |
| 未找到 HTTP 代理 | WrapListener 返回原始 listener，无 frpc 启动 |
| 配置加载失败 | Provision 阶段返回错误，Caddy 停止 |
| 工作连接错误 | 记录日志，不崩溃，等待下一个连接 |
| frps 断开重连 | frpc 库 `keepControllerWorking()` 自动处理 |

## 8. 文件清单

| 文件 | 职责 |
|------|------|
| `frpc.go` | 主模块：caddy.listeners.frpc；Provision、WrapListener、CaddyModule |
| `config.go` | 配置加载：检测格式、解析 frpc.toml/ini、提取 HTTP 代理列表 |
| `listener.go` | frpcListener 适配器：workConn channel → net.Listener |
| `http_proxy.go` | HTTP 隧道子模块：caddy.listeners.frpc.tunnels.http |
| `https_proxy.go` | HTTPS 隧道子模块：caddy.listeners.frpc.tunnels.https |
| `caddyfile.go` | Caddyfile 解析 |
| `frpc_test.go` | 单元测试 |

## 9. 与参考项目的差异

| 维度 | caddy-ngrok-listener | caddy-frpc |
|------|---------------------|------------|
| 隧道源 | ngrok-go SDK 直接返回 listener | frpc 库 + workConn → Listener 适配层 |
| 子模块 | tcp/http/tls/oauth/oidc | http/https (frp proxy types) |
| 配置 | Caddyfile/JSON 内联 | frpc.toml/ini 独立文件或内联 |
| 生命周期 | ngrok session = listener 生命周期 | frpc Service = 后台 goroutine |
| TLS 处理 | ngrok 边缘处理 | frps 处理或 Caddy 处理 |

## 10. 依赖

- `github.com/caddyserver/caddy/v2` ≥ v2.7
- `github.com/fatedier/frp` (dev 分支 latest)
- Go 1.21+
- xcaddy 构建工具

## 11. 构建

```bash
# 开发构建
xcaddy build --with github.com/hxgm/caddy-frpc

# 指定 Caddy 版本
xcaddy build v2.7.6 --with github.com/hxgm/caddy-frpc
```

## 12. 测试策略

- **单元测试:** frpcListener Accept/Close 行为
- **集成测试:** 使用 frps (Docker) 启动隧道，验证 Caddy 可处理请求
- **配置解析测试:** TOML/INI 加载 + 过滤

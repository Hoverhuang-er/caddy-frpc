# Task 4: Main Module (frpc.go)

**Files:**
- Create: `frpc.go`
- Do NOT modify any other files

**Important: Use `log/slog` (standard library), NOT `go.uber.org/zap`.**

## Implementation

Create `frpc.go`:

```go
package frpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/policy/security"
)

func init() {
	caddy.RegisterModule(new(FRPC))
}

// FRPC is a Caddy listener_wrapper that embeds frpc.
type FRPC struct {
	// ConfigFile is the path to the frpc configuration file.
	ConfigFile string `json:"config_file,omitempty"`

	frpcCfg   *frpcConfig
	listeners map[string]*frpcListener
	svr       *client.Service
	cancel    context.CancelFunc
	ctx       context.Context
	mu        sync.Mutex
}

// CaddyModule returns the module information.
func (FRPC) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.listeners.frpc",
		New: func() caddy.Module {
			return new(FRPC)
		},
	}
}

// Provision implements caddy.Provisioner.
func (f *FRPC) Provision(ctx caddy.Context) error {
	f.listeners = make(map[string]*frpcListener)
	f.ctx = context.Background()

	if f.ConfigFile == "" {
		return fmt.Errorf("frpc config_file is required")
	}

	cfg, err := loadConfig(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("loading frpc config: %w", err)
	}
	f.frpcCfg = cfg

	total := len(cfg.HTTPProxies) + len(cfg.HTTPSProxies)
	if total == 0 {
		slog.Warn("no HTTP/HTTPS proxies found in frpc config; frpc will not start")
		return nil
	}

	for _, p := range cfg.HTTPProxies {
		addr := fmt.Sprintf(":%d", p.LocalPort)
		ln := newFRPCListener(p.Name, addr)
		f.listeners[p.Name] = ln
		slog.Info("registered HTTP proxy listener",
			"proxy", p.Name, "addr", addr, "domains", p.CustomDomains)
	}
	for _, p := range cfg.HTTPSProxies {
		addr := fmt.Sprintf(":%d", p.LocalPort)
		ln := newFRPCListener(p.Name, addr)
		f.listeners[p.Name] = ln
		slog.Info("registered HTTPS proxy listener",
			"proxy", p.Name, "addr", addr, "domains", p.CustomDomains)
	}

	return nil
}

// WrapListener implements caddy.ListenerWrapper by starting the frpc service
// and returning a multiListener that yields work connections from frps.
func (f *FRPC) WrapListener(_ net.Listener) net.Listener {
	if len(f.listeners) == 0 {
		ln := newFRPCListener("default", ":0")
		close(ln.ConnChan())
		return ln
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	cfg := f.frpcCfg
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(proxyConfigurers(cfg), nil); err != nil {
		slog.Error("failed to set config source", "error", err)
		return f.fallbackListener()
	}

	aggregator := source.NewAggregator(configSource)
	proxyCfgs, _, err := aggregator.Load()
	if err != nil {
		slog.Error("aggregator load failed", "error", err)
		return f.fallbackListener()
	}
	proxyCfgs, _ = config.FilterClientConfigurers(cfg.Common, proxyCfgs, nil)
	proxyCfgs = config.CompleteProxyConfigurers(proxyCfgs)

	warning, err := validation.ValidateAllClientConfig(cfg.Common, proxyCfgs, nil, &security.UnsafeFeatures{})
	if warning != nil {
		slog.Warn("frpc config validation warning", "warning", warning)
	}
	if err != nil {
		slog.Error("frpc config validation error", "error", err)
		return f.fallbackListener()
	}

	ctx, cancel := context.WithCancel(f.ctx)
	f.cancel = cancel

	handleCb := func(baseCfg *v1.ProxyBaseConfig, conn net.Conn, m *msg.StartWorkConn) bool {
		ln, ok := f.listeners[baseCfg.Name]
		if !ok {
			slog.Warn("received work conn for unknown proxy", "name", baseCfg.Name)
			return true
		}
		select {
		case ln.ConnChan() <- conn:
			return false
		default:
			slog.Error("listener channel full, dropping work conn", "proxy", baseCfg.Name)
			conn.Close()
			return false
		}
	}

	svr, err := client.NewService(client.ServiceOptions{
		Common:                 cfg.Common,
		ConfigSourceAggregator: aggregator,
		ConfigFilePath:         f.ConfigFile,
		UnsafeFeatures:         &security.UnsafeFeatures{},
		HandleWorkConnCb:       handleCb,
	})
	if err != nil {
		slog.Error("failed to create frpc service", "error", err)
		cancel()
		return f.fallbackListener()
	}
	f.svr = svr

	go func() {
		if err := svr.Run(ctx); err != nil {
			slog.Error("frpc service exited", "error", err)
		}
	}()

	return newMultiListener(f.listeners)
}

// fallbackListener returns a closed listener when frpc fails to start.
func (f *FRPC) fallbackListener() net.Listener {
	ln := newFRPCListener("error", ":0")
	close(ln.ConnChan())
	return ln
}

// Cleanup implements caddy.CleanerUpper.
func (f *FRPC) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancel != nil {
		f.cancel()
	}
	if f.svr != nil {
		f.svr.GracefulClose(0)
	}
	for _, ln := range f.listeners {
		ln.Close()
	}
	return nil
}

// proxyConfigurers converts HTTP/HTTPS configs back to ProxyConfigurer slice.
func proxyConfigurers(cfg *frpcConfig) []v1.ProxyConfigurer {
	out := make([]v1.ProxyConfigurer, 0, len(cfg.HTTPProxies)+len(cfg.HTTPSProxies))
	for _, p := range cfg.HTTPProxies {
		p := p
		out = append(out, &p)
	}
	for _, p := range cfg.HTTPSProxies {
		p := p
		out = append(out, &p)
	}
	return out
}

// Ensure interface compliance
var (
	_ caddy.Module          = (*FRPC)(nil)
	_ caddy.Provisioner     = (*FRPC)(nil)
	_ caddy.ListenerWrapper = (*FRPC)(nil)
	_ caddy.CleanerUpper    = (*FRPC)(nil)
)
```

**Important notes on the interfaces:**
- `caddy.ListenerWrapper` requires `WrapListener(net.Listener) net.Listener`
- `caddy.Module` requires `CaddyModule() caddy.ModuleInfo`
- `caddy.Provisioner` requires `Provision(caddy.Context) error`
- `caddy.CleanerUpper` requires `Cleanup() error`

## Acceptance

- `go build ./...` succeeds
- `go vet ./...` passes
- All 7 existing tests still pass
- No import of go.uber.org/zap in our code
- Committed with message `"feat: add main FRPC module with Provision/WrapListener"`

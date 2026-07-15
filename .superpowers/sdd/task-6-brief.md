# Task 6: Refactor to Visitor Mode (config.go + frpc.go)

## What Changed

The architecture switched from `listener_wrapper` (proxies) to `caddy.app` (visitors) mode.

**Old design:**
- Module: `caddy.listeners.frpc` (ListenerWrapper)
- Config: loads `[[proxies]]`, filters HTTP/HTTPS
- WrapListener starts frpc + multiListener

**New design:**
- Module: `caddy.apps.frpc` (caddy.App)
- Config: loads `[[visitors]]` from frpc.toml
- frpc Service starts in background, creates visitor listeners on bindAddr:bindPort
- Caddy's HTTP server runs independently; can reverse_proxy to visitor ports

## Files to Modify

### config.go

Replace `frpcConfig`:

```go
package frpc

import (
	"fmt"
	"log/slog"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// frpcConfig holds parsed common config and visitor configs.
type frpcConfig struct {
	Common   *v1.ClientCommonConfig
	Visitors []v1.VisitorConfigurer
}

// loadConfig loads a frpc config file (TOML or INI), returning common client
// config and visitor configs.
func loadConfig(path string) (*frpcConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("frpc config path is empty")
	}

	result, err := config.LoadClientConfigResult(path, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load frpc config from %s: %w", path, err)
	}

	slog.Info("loaded frpc config",
		"server", result.Common.ServerAddr,
		"proxies", len(result.Proxies),
		"visitors", len(result.Visitors))

	cfg := &frpcConfig{
		Common:   result.Common,
		Visitors: result.Visitors,
	}

	for _, v := range cfg.Visitors {
		base := v.GetBaseConfig()
		slog.Info("loaded visitor", "name", base.Name, "type", base.Type,
			"serverName", base.ServerName, "bindAddr", base.BindAddr, "bindPort", base.BindPort)
	}

	return cfg, nil
}
```

### frpc.go

Rewrite as `caddy.App` module:

```go
package frpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
)

func init() {
	caddy.RegisterModule(new(FRPC))
}

// FRPC is a Caddy app module that runs frpc in visitor mode.
type FRPC struct {
	// ConfigFile is the path to the frpc configuration file.
	ConfigFile string `json:"config_file,omitempty"`

	frpcCfg *frpcConfig
	svr     *client.Service
	cancel  context.CancelFunc
	ctx     context.Context
	mu      sync.Mutex
}

// CaddyModule returns the module information.
func (FRPC) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.apps.frpc",
		New: func() caddy.Module {
			return new(FRPC)
		},
	}
}

// Provision implements caddy.Provisioner.
func (f *FRPC) Provision(ctx caddy.Context) error {
	f.ctx = context.Background()

	if f.ConfigFile == "" {
		return fmt.Errorf("frpc config_file is required")
	}

	cfg, err := loadConfig(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("loading frpc config: %w", err)
	}
	f.frpcCfg = cfg

	if len(cfg.Visitors) == 0 {
		slog.Warn("no visitors found in frpc config; frpc will not start")
		return nil
	}

	return nil
}

// Start implements caddy.App.
func (f *FRPC) Start() error {
	if f.frpcCfg == nil || len(f.frpcCfg.Visitors) == 0 {
		slog.Warn("no visitors configured, frpc app not starting")
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	cfg := f.frpcCfg

	// Build config source with visitors
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(nil, cfg.Visitors); err != nil {
		return fmt.Errorf("failed to set config source: %w", err)
	}

	aggregator := source.NewAggregator(configSource)

	proxyCfgs, visitorCfgs, err := aggregator.Load()
	if err != nil {
		return fmt.Errorf("aggregator load failed: %w", err)
	}
	// Visitors are the primary config; proxies may exist for STCP matching
	proxyCfgs, visitorCfgs = config.FilterClientConfigurers(cfg.Common, proxyCfgs, visitorCfgs)
	proxyCfgs = config.CompleteProxyConfigurers(proxyCfgs)
	visitorCfgs = config.CompleteVisitorConfigurers(visitorCfgs)

	warning, err := validation.ValidateAllClientConfig(cfg.Common, proxyCfgs, visitorCfgs, &security.UnsafeFeatures{})
	if warning != nil {
		slog.Warn("frpc config validation warning", "warning", warning)
	}
	if err != nil {
		return fmt.Errorf("frpc config validation failed: %w", err)
	}

	ctx, cancel := context.WithCancel(f.ctx)
	f.cancel = cancel

	svr, err := client.NewService(client.ServiceOptions{
		Common:                 cfg.Common,
		ConfigSourceAggregator: aggregator,
		ConfigFilePath:         f.ConfigFile,
		UnsafeFeatures:         &security.UnsafeFeatures{},
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create frpc service: %w", err)
	}
	f.svr = svr

	go func() {
		slog.Info("frpc visitor service starting",
			"server", cfg.Common.ServerAddr,
			"visitors", len(cfg.Visitors))
		if err := svr.Run(ctx); err != nil {
			slog.Error("frpc visitor service exited", "error", err)
		}
	}()

	return nil
}

// Stop implements caddy.App.
func (f *FRPC) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cancel != nil {
		f.cancel()
	}
	if f.svr != nil {
		f.svr.GracefulClose(0)
	}
	return nil
}

// Cleanup implements caddy.CleanerUpper.
func (f *FRPC) Cleanup() error {
	return f.Stop()
}

// Interface assertions
var (
	_ caddy.Module          = (*FRPC)(nil)
	_ caddy.Provisioner     = (*FRPC)(nil)
	_ caddy.App             = (*FRPC)(nil)
	_ caddy.CleanerUpper    = (*FRPC)(nil)
)
```

### caddyfile.go

Update UnmarshalCaddyfile — syntax stays the same (`frpc <config_path>`), just the context changes.

## Tests

Update `frpc_test.go`:
- Remove or update tests that reference `caddy.listeners.frpc` and proxy-specific features
- Keep listener tests (frpcListener, multiListener) — they're generic
- Update config tests to verify visitors instead of proxies
- Add a test for module ID: `"caddy.apps.frpc"`

## Acceptance

- `go build ./...` succeeds
- `go vet ./...` passes
- Updated tests pass
- Module ID is `caddy.apps.frpc`
- Committed

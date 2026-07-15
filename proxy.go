package frpc

import (
	"context"
	"encoding/json"
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

// FRPC is a Caddy app that embeds an frpc client in visitor mode.
type FRPC struct {
	// ConfigFile is the path to the frpc configuration file.
	ConfigFile string `json:"config_file,omitempty"`

	// ConfigRaw can be used instead of ConfigFile to embed frpc configuration
	// inline in Caddy JSON. Supported formats: TOML, YAML, JSON, INI.
	ConfigRaw json.RawMessage `json:"config,omitempty"`

	frpcCfg *frpcConfig
	svr     *client.Service
	cancel  context.CancelFunc
	ctx     context.Context
	mu      sync.Mutex
}

// CaddyModule returns the module information.
func (f *FRPC) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.apps.frpc",
		New: func() caddy.Module {
			return new(FRPC)
		},
	}
}
// Provision implements caddy.Provisioner.
func (f *FRPC) Provision(ctx caddy.Context) error {

	if f.svr != nil {
		return fmt.Errorf("frpc already provisioned")
	}

	f.ctx = ctx

	var cfg *frpcConfig
	var err error

	if f.ConfigRaw != nil {
		cfg, err = loadConfigFromBytes(f.ConfigRaw, "auto")
	} else if f.ConfigFile != "" {
		cfg, err = loadConfig(f.ConfigFile)
	} else {
		return fmt.Errorf("frpc config required: set config_file or config inline")
	}
	if err != nil {
		return fmt.Errorf("loading frpc config: %w", err)
	}
	f.frpcCfg = cfg

	if len(cfg.Visitors) == 0 {
		slog.Warn("no visitors found in frpc config; frpc will not accept connections")
	}

	return nil
}

// Start implements caddy.App by launching the frpc client service.
func (f *FRPC) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	cfg := f.frpcCfg
	if cfg == nil {
		return fmt.Errorf("frpc not provisioned")
	}

	if len(cfg.Visitors) == 0 {
		slog.Warn("no visitors configured, frpc app not starting")
		return nil
	}

	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(nil, cfg.Visitors); err != nil {
		return fmt.Errorf("setting config source: %w", err)
	}

	aggregator := source.NewAggregator(configSource)
	proxyCfgs, visitorCfgs, err := aggregator.Load()
	if err != nil {
		return fmt.Errorf("aggregator load: %w", err)
	}
	proxyCfgs, visitorCfgs = config.FilterClientConfigurers(cfg.Common, proxyCfgs, visitorCfgs)
	proxyCfgs = config.CompleteProxyConfigurers(proxyCfgs)
	visitorCfgs = config.CompleteVisitorConfigurers(visitorCfgs)

	warning, err := validation.ValidateAllClientConfig(cfg.Common, proxyCfgs, visitorCfgs, &security.UnsafeFeatures{})
	if warning != nil {
		slog.Warn("frpc config validation warning", "warning", warning)
	}
	if err != nil {
		return fmt.Errorf("config validation: %w", err)
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
		f.cancel = nil
		return fmt.Errorf("creating frpc service: %w", err)
	}
	f.svr = svr

	go func() {
		if err := svr.Run(ctx); err != nil {
			slog.Error("frpc service exited", "error", err)
		}
	}()

	return nil
}

// Stop implements caddy.App by gracefully stopping the frpc client service.
func (f *FRPC) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	if f.svr != nil {
		f.svr.GracefulClose(0)
		f.svr = nil
	}
	return nil
}

// Cleanup implements caddy.CleanerUpper.
func (f *FRPC) Cleanup() error {
	return f.Stop()
}

// Ensure interface compliance
var (
	_ caddy.Module       = (*FRPC)(nil)
	_ caddy.Provisioner  = (*FRPC)(nil)
	_ caddy.App          = (*FRPC)(nil)
	_ caddy.CleanerUpper = (*FRPC)(nil)
)

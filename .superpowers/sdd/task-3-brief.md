# Task 3: Config Loading (config.go)

**Files:**
- Create: `config.go`
- Modify: `frpc_test.go` (add config tests)

**Important: Use `log/slog` (standard library), NOT `go.uber.org/zap`.**

## Implementation

Create `config.go`:

```go
package frpc

import (
	"fmt"
	"log/slog"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config"
)

// frpcConfig holds parsed common config and filtered HTTP/HTTPS proxies.
type frpcConfig struct {
	Common       *v1.ClientCommonConfig
	HTTPProxies  []v1.HTTPProxyConfig
	HTTPSProxies []v1.HTTPSProxyConfig
}

// loadConfig loads a frpc config file (TOML or INI), returning common client
// config and only HTTP/HTTPS proxy configs. Other proxy types are logged and skipped.
func loadConfig(path string) (*frpcConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("frpc config path is empty")
	}

	result, err := config.LoadClientConfigResult(path, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load frpc config from %s: %w", path, err)
	}

	cfg := &frpcConfig{
		Common: result.Common,
	}

	for _, pc := range result.Proxies {
		base := pc.GetBaseConfig()
		switch base.Type {
		case "http":
			if hc, ok := pc.(*v1.HTTPProxyConfig); ok {
				hc.Complete()
				cfg.HTTPProxies = append(cfg.HTTPProxies, *hc)
			}
		case "https":
			if hc, ok := pc.(*v1.HTTPSProxyConfig); ok {
				hc.Complete()
				cfg.HTTPSProxies = append(cfg.HTTPSProxies, *hc)
			}
		default:
			slog.Warn("skipping non-HTTP proxy", "name", base.Name, "type", base.Type)
		}
	}

	return cfg, nil
}
```

## Tests (append to frpc_test.go)

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigTOML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "frpc.toml")
	content := `
serverAddr = "frps.example.com"
serverPort = 7000
auth.token = "secret"

[[proxies]]
name = "web"
type = "http"
localIP = "127.0.0.1"
localPort = 8080
customDomains = ["web.example.com"]
locations = ["/"]

[[proxies]]
name = "api"
type = "http"
localIP = "127.0.0.1"
localPort = 9000

[[proxies]]
name = "ssh-tunnel"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000

[[proxies]]
name = "secure"
type = "https"
localIP = "127.0.0.1"
localPort = 443
customDomains = ["secure.example.com"]
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if cfg.Common.ServerAddr != "frps.example.com" {
		t.Fatalf("expected frps.example.com, got %s", cfg.Common.ServerAddr)
	}
	if cfg.Common.ServerPort != 7000 {
		t.Fatalf("expected 7000, got %d", cfg.Common.ServerPort)
	}
	if len(cfg.HTTPProxies) != 2 {
		t.Fatalf("expected 2 HTTP proxies, got %d", len(cfg.HTTPProxies))
	}
	if cfg.HTTPProxies[0].Name != "web" {
		t.Fatalf("expected proxy name web, got %s", cfg.HTTPProxies[0].Name)
	}
	if len(cfg.HTTPSProxies) != 1 {
		t.Fatalf("expected 1 HTTPS proxy, got %d", len(cfg.HTTPSProxies))
	}
}

func TestLoadConfigNoHTTPProxy(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "frpc.toml")
	content := `
serverAddr = "frps.example.com"
serverPort = 7000

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if len(cfg.HTTPProxies)+len(cfg.HTTPSProxies) != 0 {
		t.Fatalf("expected 0 http/https proxies, got %d http + %d https",
			len(cfg.HTTPProxies), len(cfg.HTTPSProxies))
	}
	if cfg.Common.ServerAddr != "frps.example.com" {
		t.Fatalf("expected frps.example.com, got %s", cfg.Common.ServerAddr)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/frpc.toml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
```

## Acceptance

- All 3 config tests PASS (plus existing listener tests)
- `go vet ./...` passes
- No import of go.uber.org/zap anywhere in our own code
- Committed with message `"feat: add config loader (frpc.toml/ini parsing)"`

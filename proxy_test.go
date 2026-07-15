package frpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caddyserver/caddy/v2"
)

func TestFRPCModuleID(t *testing.T) {
	mod, err := caddy.GetModule("caddy.apps.frpc")
	if err != nil {
		t.Fatalf("module caddy.apps.frpc not registered: %v", err)
	}
	if mod.ID != "caddy.apps.frpc" {
		t.Fatalf("expected module ID caddy.apps.frpc, got %s", mod.ID)
	}
}
func TestLoadConfigWithVisitor(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "frpc.toml")
	content := `
serverAddr = "frps.example.com"
serverPort = 7000
auth.token = "secret"

[[visitors]]
name = "dashboard"
type = "stcp"
bindAddr = "127.0.0.1"
bindPort = 7500
serverUser = "admin"
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
	if len(cfg.Visitors) != 1 {
		t.Fatalf("expected 1 visitor, got %d", len(cfg.Visitors))
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/frpc.toml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadConfigYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "frpc.yaml")
	content := `
serverAddr: frps.example.com
serverPort: 7000
auth:
  token: secret

visitors:
  - name: dashboard
    type: stcp
    bindAddr: 127.0.0.1
    bindPort: 7500
    serverUser: admin
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
	if len(cfg.Visitors) != 1 {
		t.Fatalf("expected 1 visitor, got %d", len(cfg.Visitors))
	}
}
func TestLoadConfigJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "frpc.json")
	content := `{
	"serverAddr": "frps.example.com",
	"serverPort": 7000,
	"auth": {
		"token": "secret"
	},
	"visitors": [
		{
			"name": "dashboard",
			"type": "stcp",
			"bindAddr": "127.0.0.1",
			"bindPort": 7500,
			"serverUser": "admin"
		}
	]
}`
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
	if len(cfg.Visitors) != 1 {
		t.Fatalf("expected 1 visitor, got %d", len(cfg.Visitors))
	}
}
func TestLoadConfigFromBytes(t *testing.T) {
	jsonContent := `{
	"serverAddr": "frps.example.com",
	"serverPort": 7000,
	"auth": {
		"token": "secret"
	},
	"visitors": [
		{
			"name": "dashboard",
			"type": "stcp",
			"bindAddr": "127.0.0.1",
			"bindPort": 7500,
			"serverUser": "admin"
		}
	]
}`

	cfg, err := loadConfigFromBytes([]byte(jsonContent), "auto")
	if err != nil {
		t.Fatalf("loadConfigFromBytes() error: %v", err)
	}
	if cfg.Common.ServerAddr != "frps.example.com" {
		t.Fatalf("expected frps.example.com, got %s", cfg.Common.ServerAddr)
	}
	if cfg.Common.ServerPort != 7000 {
		t.Fatalf("expected 7000, got %d", cfg.Common.ServerPort)
	}
	if len(cfg.Visitors) != 1 {
		t.Fatalf("expected 1 visitor, got %d", len(cfg.Visitors))
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		data   string
		expect string
	}{
		{`{"serverAddr":"test"}`, "json"},
		{`serverAddr: test`, "yaml"},
		{`[common]
serverAddr = test`, "ini"},
		{`serverAddr = "test"
[[visitors]]
name = "test"`, "toml"},
		{``, "toml"},
	}

	for _, tt := range tests {
		got := detectFormat([]byte(tt.data))
		if got != tt.expect {
			t.Errorf("detectFormat(%q) = %q, want %q", tt.data, got, tt.expect)
		}
	}
}

func TestLoadConfigFromBytesInvalid(t *testing.T) {
	_, err := loadConfigFromBytes([]byte(`not valid config`), "toml")
	if err == nil {
		t.Fatal("expected error for invalid config content")
	}
}

func TestLoadExampleTOML(t *testing.T) {
	cfg, err := loadConfig("examples/frpc.toml")
	if err != nil {
		t.Fatalf("failed to load example frpc.toml: %v", err)
	}
	if len(cfg.Visitors) == 0 {
		t.Fatal("expected at least one visitor from example frpc.toml")
	}
	if cfg.Common.ServerAddr != "frps.example.com" {
		t.Fatalf("expected frps.example.com, got %s", cfg.Common.ServerAddr)
	}
}

func TestLoadExampleINI(t *testing.T) {
	cfg, err := loadConfig("examples/frpc.ini")
	if err != nil {
		t.Fatalf("failed to load example frpc.ini: %v", err)
	}
	if len(cfg.Visitors) == 0 {
		t.Fatal("expected at least one visitor from example frpc.ini")
	}
}

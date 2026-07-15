package frpc

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config"
)

// frpcConfig holds parsed common config and visitors.
type frpcConfig struct {
	Common   *v1.ClientCommonConfig
	Visitors []v1.VisitorConfigurer
}

// loadConfig loads a frpc config file (TOML, YAML, JSON, or INI), returning common
// client config and visitor configs. Proxy configs are logged and skipped.
func loadConfig(path string) (*frpcConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("frpc config path is empty")
	}

	result, err := config.LoadClientConfigResult(path, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load frpc config from %s: %w", path, err)
	}

	cfg := &frpcConfig{
		Common:   result.Common,
		Visitors: result.Visitors,
	}

	for _, pc := range result.Proxies {
		base := pc.GetBaseConfig()
		slog.Warn("skipping non-visitor proxy config; visitor mode does not use proxies",
			"name", base.Name, "type", base.Type)
	}

	for _, v := range result.Visitors {
		base := v.GetBaseConfig()
		slog.Info("loaded visitor", "name", base.Name, "type", base.Type, "serverName", base.ServerName, "bindAddr", base.BindAddr, "bindPort", base.BindPort)
	}

	return cfg, nil
}

// loadConfigFromBytes loads frpc configuration from raw bytes using the specified
// format ("toml", "yaml", "json", "ini", or "auto" for automatic detection).
// It writes the data to a temp file with the correct extension, then delegates to
// config.LoadClientConfigResult.
func loadConfigFromBytes(data []byte, format string) (*frpcConfig, error) {
	if format == "auto" {
		format = detectFormat(data)
	}

	ext := "." + format
	tmp, err := os.CreateTemp("", "frpc-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("failed to write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp config: %w", err)
	}

	return loadConfig(tmpPath)
}

// detectFormat attempts to detect the frpc config format from raw bytes using
// simple heuristics. Returns "json", "yaml", "ini", or "toml" (default).
func detectFormat(data []byte) string {
	s := strings.TrimSpace(string(data))
	if len(s) == 0 {
		return "toml"
	}

	// JSON typically starts with '{'
	first := s[0]
	if first == '{' {
		return "json"
	}

	// INI-style: [section] headers at the start of a line
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// [[proxies]] / [[visitors]] is TOML array-of-tables
			if strings.HasPrefix(trimmed, "[[") {
				return "toml"
			}
			return "ini"
		}
	}

	// YAML: key: value pairs (frpc config uses this style)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, ": ") || strings.Contains(trimmed, ":\t") {
			return "yaml"
		}
	}

	// Default to TOML
	return "toml"
}

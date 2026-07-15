# Multi-Format Config Support + Inline Config Injection

## What was done

### 1. config.go — inline config loader + format detection

- Added `loadConfigFromBytes(data []byte, format string) (*frpcConfig, error)` — writes raw bytes to a temp file with the correct extension, delegates to `config.LoadClientConfigResult`, and cleans up.
- Added `detectFormat(data []byte) string` — simple heuristic detection:
  - Starts with `{` → JSON
  - `[section]` header (not `[[`) → INI
  - `[[proxies]]`/`[[visitors]]` → TOML
  - `key: value` / `key:\tvalue` pattern → YAML
  - Default → TOML
- Updated `loadConfig` doc comment to document all supported formats (TOML/YAML/JSON/INI).

### 2. frpc.go — ConfigRaw field + branching Provision

- Added `ConfigRaw json.RawMessage` field with tag `json:"config,omitempty"`.
- Updated `Provision()` to try `ConfigRaw` first, then `ConfigFile`, then error: `"frpc config required: set config_file or config inline"`.
- Added `encoding/json` import.

### 3. caddyfile.go — block subdirective support

- Original: `frpc <path>` (single argument).
- New: also supports block syntax:
  ```
  frpc {
      config ./frpc.yaml
  }
  ```

### 4. frpc_test.go — multi-format tests

| Test | What it covers |
|---|---|
| `TestLoadConfigYAML` | `.yaml` file with nested `auth: { token }` |
| `TestLoadConfigJSON` | `.json` file with nested `"auth": {"token"}` |
| `TestLoadConfigFromBytes` | `loadConfigFromBytes` with JSON, format="auto" |
| `TestDetectFormat` | format detection heuristics (5 cases) |
| `TestLoadConfigFromBytesInvalid` | error on garbage content |

## Verification

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test -v -count=1 ./...` — all 9 tests PASS (4 existing + 5 new)

## Commit

`0c1b8f7` — `feat: support TOML/YAML/JSON/INI config formats and inline config injection`

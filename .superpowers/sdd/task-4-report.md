# Task 4 Report: Main Module (frpc.go)

## Status: Complete

## Files Created/Modified
- **Created:** `frpc.go` — Main FRPC module implementing caddy.ListenerWrapper
- **Modified:** `config.go` — Added `Visitors` field to `frpcConfig`, populated from `result.Visitors`
- **Modified:** `listener.go` — Added `newFRPCListenerBuf()` constructor with configurable buffer size
- **Modified:** `go.mod`, `go.sum` — Upgraded caddy from v2.7.6 to v2.11.4 (required for Go 1.26 compatibility)

## Caddy Version
Upgraded from v2.7.6 to v2.11.4 because v2.7.6 uses an outdated quic-go API (`http3.QUICEarlyListener`, `quic.EarlyConnection`) that doesn't compile with Go 1.26. All core interfaces (`caddy.Module`, `caddy.Provisioner`, `caddy.ListenerWrapper`, `caddy.CleanerUpper`) are stable across this upgrade.

## Design Decisions

### Visitors
- `frpcConfig.Visitors` holds `[]v1.VisitorConfigurer` from the config load result
- Passed through to `ReplaceAll()`, `FilterClientConfigurers()`, `CompleteVisitorConfigurers()`, and `ValidateAllClientConfig()`
- Only HTTP/HTTPS proxies get listeners; visitors (STCP, XTCP, SUDP) are handled by the frpc `Service` internally

### Pool Count / Listener Buffer
- New `listenerBufferSize()` helper: returns `max(64, cfg.Common.Transport.PoolCount)`
- `Provision()` uses `newFRPCListenerBuf()` with this buffer size instead of the hardcoded 8
- `newFRPCListenerBuf` was added to `listener.go` alongside the original `newFRPCListener` (kept for backward compat with tests and fallback path)

### Multi-Goroutine Safety
- `multiListener.Accept()` uses a `select` on `ml.ch` and `ml.done` — channels are inherently goroutine-safe
- Multiple goroutines can call `Accept()` concurrently; each receives the next available connection
- No mutex needed in the hot path

### Module Interfaces
- `CaddyModule()` uses pointer receiver `(*FRPC)` to avoid `go vet` warning about copying `sync.Mutex`
- All four interface compliance checks: `caddy.Module`, `caddy.Provisioner`, `caddy.ListenerWrapper`, `caddy.CleanerUpper`

## Verification
- `go build ./...` — passes
- `go vet ./...` — passes (no lock-copy warnings)
- `go test -count=1 ./...` — all 7 tests pass
- No `go.uber.org/zap` imports in our code
- Committed as `d7b2d70` with message "feat: add main FRPC module with Provision/WrapListener"

# Task 6: Refactor to Visitor Mode (caddy.App)

## Summary

Refactored caddy-frpc from `caddy.listeners.frpc` (ListenerWrapper) to `caddy.apps.frpc` (caddy.App) visitor mode.

## Changes

### config.go
- Removed `HTTPProxies` and `HTTPSProxies` fields from `frpcConfig`
- Simplified `loadConfig` to only retain `Common` config and `Visitors`
- Proxy configs are now logged with a warning and skipped (visitor mode does not use them)

### frpc.go
- Module ID changed from `"caddy.listeners.frpc"` to `"caddy.apps.frpc"`
- Removed `listeners` map, `listenerBufferSize()`, `fallbackListener()`, and `proxyConfigurers()` — no longer needed
- Replaced `WrapListener(net.Listener) net.Listener` with `Start() error` and `Stop() error` (caddy.App interface)
- `Provision` now only loads config (no listener setup)
- `Start` creates the frpc service with visitor configurers and runs it in a goroutine
- `Stop` cancels the context and gracefully closes the service
- `Cleanup` delegates to `Stop`
- `FRPC` struct simplified: removed `listeners` map
- Interface assertions updated: removed `caddy.ListenerWrapper`, added `caddy.App`

### caddyfile.go
- No functional changes (doc comment already generic enough)

### frpc_test.go
- Added `TestFRPCModuleID` checking `"caddy.apps.frpc"`
- Added `TestLoadConfigWithVisitor` testing visitor config loading
- Removed `TestLoadConfigTOML` and `TestLoadConfigNoHTTPProxy` (proxy-specific)
- Kept `TestLoadConfigNotFound`, `TestFRPCListenerAcceptClose`, `TestFRPCListenerCloseUnblocksAccept`, `TestFRPCListenerName`, `TestMultiListener`

## Verification

- `go build ./...` — passes
- `go test -count=1 ./...` — all tests pass
- `go vet ./...` — clean

## Files preserved without modification

- `listener.go` (frpcListener, multiListener kept for future proxy support)

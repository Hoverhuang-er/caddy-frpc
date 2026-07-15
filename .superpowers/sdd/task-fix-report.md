# Task Fix Report

## Changes

### Fix 1: Remove dead code
- **listener.go** (deleted): Removed `frpcListener`, `multiListener`, and `resolveAddr` — all dead code with no references in config.go, frpc.go, or caddyfile.go.

### Fix 2: Remove dead listener tests
- **frpc_test.go**: Removed 4 tests that tested the deleted listener types:
  - `TestFRPCListenerAcceptClose`
  - `TestFRPCListenerCloseUnblocksAccept`
  - `TestFRPCListenerName`
  - `TestMultiListener`
- Also removed unused `"net"` import.

### Fix 3: Add Provision idempotency guard
- **frpc.go**: Added guard at the start of `Provision()`:
  ```go
  if f.svr != nil {
      return fmt.Errorf("frpc already provisioned")
  }
  ```

## Verification
- `go build ./...` -- OK
- `go vet ./...` -- OK
- `go test -v -count=1 ./...` -- all 3 remaining tests pass:
  - `TestFRPCModuleID` -- PASS
  - `TestLoadConfigWithVisitor` -- PASS
  - `TestLoadConfigNotFound` -- PASS

## Commit
`0a14f2d` — `chore: remove dead proxy listener code, add Provision guard`

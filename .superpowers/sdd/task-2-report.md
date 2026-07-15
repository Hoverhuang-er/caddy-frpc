# Task 2: Listener Adapter — Report

## Status: DONE

## Commits
- `cc3f904` — "feat: add frpcListener adapter (workConn -> net.Listener)"

## Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `listener.go` | 111 | `frpcListener` (workConn channel → `net.Listener`) + `multiListener` (fan-in multiple proxy listeners) + `resolveAddr` helper |
| `frpc_test.go` | 89 | 4 tests covering frpcListener and multiListener |

## Test Results (go test -v -count=1)

| Test | Result |
|------|--------|
| `TestFRPCListenerAcceptClose` | PASS — pushes a piped conn through ConnChan, Accept() receives it, verifies Addr() |
| `TestFRPCListenerCloseUnblocksAccept` | PASS — close unblocks pending Accept, returns `net.ErrClosed` |
| `TestFRPCListenerName` | PASS — Name() returns the configured proxy name |
| `TestMultiListener` | PASS — multiListener fans in from one sub-listener via Accept() |

## Verification

| Check | Result |
|-------|--------|
| `go test -v -count=1 ./...` | PASS (4/4) |
| `go vet ./...` | PASS (exit 0, no output) |

## Implementation Notes

- `resolveAddr` fills in a nil IP → `127.0.0.1` after a successful `net.ResolveTCPAddr`, so bare ports like `:8080` produce `127.0.0.1:8080` as the canonical address string rather than `:8080`. This matches the test expectation.
- The `multiListener` goroutine-per-listener pattern uses a local variable shadow (`ln := ln`) to capture each listener correctly in the closure.
- `frpcListener.Close()` uses `atomic.Bool.CompareAndSwap` for safe idempotent close; subsequent calls return `net.ErrClosed`.

## Concerns
- None. The listener adapter is self-contained with no external frp imports at this layer — it operates purely on `net.Conn` channels.

## Report written to
`.superpowers/sdd/task-2-report.md`

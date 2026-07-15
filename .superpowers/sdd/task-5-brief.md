# Task 5: Caddyfile Support (caddyfile.go)

**Files:**
- Create: `caddyfile.go`

## Implementation

Create `caddyfile.go`:

```go
package frpc

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
// Syntax:
//     frpc <config_path>
//
// Example:
//     frpc ./frpc.toml
func (f *FRPC) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&f.ConfigFile) {
			return d.ArgErr()
		}
		if d.NextArg() {
			return d.ArgErr()
		}
	}
	return nil
}

// Interface assertions
var (
	_ caddyfile.Unmarshaler = (*FRPC)(nil)
)
```

Add the import for caddyfile to the imports.

## Acceptance

- `go build ./...` succeeds
- `go vet ./...` passes
- All 7 existing tests still pass
- Committed with message `"feat: add Caddyfile unmarshaling"`

## No Tests Needed

The UnmarshalCaddyfile method is straightforward. It sets ConfigFile from the first argument. Integration-level Caddyfile parsing is covered by Caddy's own test framework.

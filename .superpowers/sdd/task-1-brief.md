# Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `Makefile`

## Steps

1. `go mod init github.com/hxgm/caddy-frpc`
2. `go get github.com/caddyserver/caddy/v2@v2.7.6`
3. `go get github.com/fatedier/frp@dev`
4. `go mod tidy`
5. Create `Makefile` with build/test/clean targets
6. Commit

## Makefile Content

```makefile
MODULE = github.com/hxgm/caddy-frpc
CADDY_VERSION ?= v2.7.6

.PHONY: build test clean

build:
	xcaddy build $(CADDY_VERSION) --with $(MODULE)

test:
	go test -v -race ./...

clean:
	rm -f caddy
```

## Acceptance

- `go.mod` created with module path `github.com/hxgm/caddy-frpc`
- `go.sum` present
- `go build ./...` succeeds
- `go vet ./...` succeeds
- Committed with message `"chore: scaffold caddy-frpc module"`

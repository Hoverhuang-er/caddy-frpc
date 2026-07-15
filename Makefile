MODULE = github.com/hxgm/caddy-frpc
CADDY_VERSION ?= v2.11.4

.PHONY: build test clean

build:
	xcaddy build $(CADDY_VERSION) --with $(MODULE)

test:
	go test -v -race ./...

clean:
	rm -f caddy

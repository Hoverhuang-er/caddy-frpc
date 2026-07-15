# syntax=docker/dockerfile:1
# Multi-stage build: compile caddy with caddy-frpc plugin,
# then copy to a minimal runtime image.

ARG CADDY_VERSION=v2.11.4
ARG GO_VERSION=1.26

# ---- Build stage ----
FROM golang:${GO_VERSION}-alpine AS builder

ARG CADDY_VERSION
ARG MODULE=github.com/Hoverhuang-er/caddy-frpc
ARG MODULE_VERSION=v0.1.1

RUN apk add --no-cache git

# Install xcaddy
RUN go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# Build caddy with the plugin
RUN xcaddy build ${CADDY_VERSION} \
    --with ${MODULE}@${MODULE_VERSION} \
    --output /usr/bin/caddy

# Verify the plugin is embedded
RUN /usr/bin/caddy list-modules | grep frpc

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates libcap

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# Set capabilities so caddy can bind to privileged ports
RUN setcap 'cap_net_bind_service=+ep' /usr/bin/caddy

EXPOSE 80 443 2019

WORKDIR /srv

ENTRYPOINT ["/usr/bin/caddy"]
CMD ["run", "--config", "/etc/caddy/Caddyfile"]

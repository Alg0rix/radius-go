# AGENTS.md — radius-go

## Current state

This repo is **not yet implemented**. Only `docs/plan.md` exists — read it first for the full design.

## Hard rules (override the plan where they conflict)

- **Every source file < 500 LOC.** Split when approaching the limit. Prefer cohesive modules over dumping grounds.
- **Modular by default.** Each package owns one concern and exposes a narrow interface. New features extend via new files/packages, not edits to a growing god-file. Adding a feature later must not regress existing packages.
- **Low regression risk.** Prefer composition + interfaces over shared mutable globals. Keep RADIUS core, repository, HTTP handlers, and config in separate packages so changes stay localized.

## Stack

- **Go 1.22+**, single module: `github.com/your-org/radius-go`
- **RADIUS**: `layeh.com/radius`
- **HTTP**: `labstack/echo/v4`
- **DB**: PostgreSQL via `jackc/pgx/v5` (pool: `pgxpool`)
- **Logging**: `rs/zerolog` (console/JSON toggle by env)
- **Config**: env vars loaded via `joho/godotenv`, validated at startup
- **Auth**: PAP only + bcrypt password hashing + HMAC-MD5 Message-Authenticator

## Architecture

Single-binary server combining a RADIUS core and an HTTP management API:
- **3 UDP PacketServers**: auth (port 1812), accounting (1813), CoA (3799)
- **HTTP server** (default 8083): management REST API + health endpoints
- **In-memory state**: NAS secrets, subscriber profiles, active sessions — refreshed from DB via ticker
- **DB**: 3 tables (`radius_users`, `radius_nas`, `radius_sessions`), migrations run at startup

## Entry point

```
cmd/api/main.go → config.Load("radius") → app.Run(cfg)
  → runtime.Bootstrap(ctx, cfg) → Dependencies{DB, Logger, Config}
  → radius.NewService(deps, cfg) → Start() (3 PacketServers + tickers)
  → echo HTTP server + graceful shutdown
```

## Key conventions (non-obvious from code)

- **All management endpoints require `X-Internal-Secret` header** via `internalsecret.Require(secret)` middleware. Health endpoints (`/health`, `/ready`, `/healthz`, `/readyz`) are public.
- **JSON envelope**: every response uses `{ success, data, meta, error }` via `runtime/jttp.go` helpers (`OK`, `Fail`, `Created`).
- **Per-NAS shared secret**: RADIUS secrets are stored per-NAS in the `radius_nas` table, looked up by remote address at auth time via `radiusSecretSource.RADIUSSecret()`.
- **Service type**: users have a `service_type` field (`framed` or `login`) — a flat toggle that avoids complex profile hierarchies.
- **MikroTik VSA** (vendor 14988): `MikroTik-Rate-Limit` and `MikroTik-Group` attributes are supported.
- **No singleton globals**: dependency injection via `Dependencies` struct, `*Service` receives `*Dependencies` + `config.Config`.
- **HA-ready**: No sticky state between requests. In-memory caches are warm copies of DB data, refreshed via ticker — a second instance can start cold and become consistent without coordination. All mutable truth lives in PostgreSQL, so multiple instances can run behind a stateless load balancer (UDP for RADIUS, HTTP for API). Assume at least 2 instances will run side by side.

## What this project deliberately excludes

- No license checks, Sentry, Redis, CHAP/MS-CHAPv2, PPPoE profiles, or isolation pool
- No `runtime.NewID()` — use `uuid.New().String()`
- No `netutil.Loopback` — use `fmt.Sprintf("127.0.0.1:%d", port)`
- Single `Config` struct (no separate appconfig alias)

## Commands

```
go build ./cmd/api            # verify build
go run ./cmd/api              # start (needs DB_DSN + INTERNAL_SECRET env)
```

## Testing with radclient

```
curl http://localhost:8083/health
curl -H "X-Internal-Secret: <secret>" http://localhost:8083/api/v1/radius/status
echo "User-Name=testuser,User-Password=testpass" | radclient -x 127.0.0.1:1812 auth testing123
```

## Design principles

- **Clarity over cleverness.** Write straightforward Go — prefer stdlib, avoid unnecessary indirection.
- **Composition over inheritance.** Interfaces define contracts; structs implement them. No deep embedding chains.
- **Each package is a self-contained concern.** The `radius` package owns the RADIUS protocol, `httpapi` owns REST routing, `config` owns env parsing — no cross-package circular dependencies.
- **Defensive defaults.** Validate config at startup. Return structured errors. Always set timeouts.

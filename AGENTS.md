# AGENTS.md — radius-go

## Current state

This repo is implemented. Use `docs/adr/` for architecture decisions and this file for conventions.

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
- **In-memory state**: NAS secrets, subscriber profiles, PPPoE profiles, voucher packages, active sessions — refreshed from DB via ticker
- **DB**: 5 tables (`radius_users`, `radius_nas`, `radius_sessions`, `voucher_packages`, `pppoe_profiles`), migrations run at startup
- **Profile model**: `service_type=login` + `pppoe_profile_id` → PPPoE user (profile provides PPP-layer RADIUS attributes + per-session caps). `service_type=framed` + `voucher_package_id` → Hotspot user (voucher package IS the hotspot profile, extended with pool/DNS). The two assignment paths are mutually exclusive.
- **Precedence**: per-user RADIUS attributes override profile defaults when set (FreeRADIUS `radreply` over `radgroupreply`); empty per-user fields fall through to the attached profile.

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
- **PPPoE profiles**: new `pppoe_profiles` entity attached via `radius_users.pppoe_profile_id`; provides PPP-layer attributes (Framed-Protocol, Framed-Pool, DNS, MTU/MRU, compression) plus per-session limits (rate-limit, bandwidth, session-timeout, idle-timeout, max-total-octets). Per-user values override profile values when set.
- **Hotspot profiles**: the existing `voucher_packages` entity serves this role, extended with `address_pool`, `primary_dns`, `secondary_dns`.
- **Mutual exclusion**: `voucher_package_id` and `pppoe_profile_id` cannot both be set on the same user.
- **MikroTik VSA** (vendor 14988): `MikroTik-Rate-Limit`, `MikroTik-Group`, `MikroTik-Address-Pool`, `MikroTik-Total-Limit` attributes are supported.
- **Microsoft VSA** (vendor 311): `MS-Primary-DNS-Server`, `MS-Secondary-DNS-Server` are emitted for DNS when configured.
- **No singleton globals**: dependency injection via `Dependencies` struct, `*Service` receives `*Dependencies` + `config.Config`.
- **HA-ready**: No sticky state between requests. In-memory caches are warm copies of DB data, refreshed via ticker — a second instance can start cold and become consistent without coordination. All mutable truth lives in PostgreSQL, so multiple instances can run behind a stateless load balancer (UDP for RADIUS, HTTP for API). Assume at least 2 instances will run side by side.

## What this project deliberately excludes

- No license checks, Sentry, Redis, CHAP/MS-CHAPv2, or isolation pool
- No `runtime.NewID()` — use `uuid.New().String()`
- No `netutil.Loopback` — use `fmt.Sprintf("127.0.0.1:%d", port)`
- Single `Config` struct (no separate appconfig alias)

## Commands

```
go build ./cmd/api            # verify build
go run ./cmd/api              # start (needs DB_DSN + INTERNAL_SECRET env)
```

## API additions

```
GET    /api/v1/pppoe-profiles
POST   /api/v1/pppoe-profiles
GET    /api/v1/pppoe-profiles/:id
PUT    /api/v1/pppoe-profiles/:id
DELETE /api/v1/pppoe-profiles/:id
```

A subscriber is assigned to a PPPoE profile via `POST /api/v1/radius/subscribers` or `PUT /api/v1/radius/subscribers/:id` with `pppoe_profile_id`.

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

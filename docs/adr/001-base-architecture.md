# ADR-003: Base RADIUS Server Architecture

## Status

Accepted

## Context

We needed a RADIUS server for ISP/hotspot use cases that could:

- Authenticate users over UDP RADIUS (PAP).
- Receive accounting start/interim/stop packets and track sessions.
- Provide a management REST API for operators.
- Persist configuration and session state across restarts.
- Run as a single binary with minimal operational complexity.

The goal was a stateless, horizontally scalable core where PostgreSQL is the
only source of truth, while in-memory caches keep the hot paths fast.

## Decision

Build a single Go binary with three responsibilities in separate packages:

1. **`radius` package** — owns the RADIUS protocol (auth, accounting, CoA),
   in-memory state, and management handlers.
2. **`httpapi` package** — owns public health endpoints and base middleware.
3. **`database` package** — owns the connection pool and migrations.

### RADIUS core

Three `layeh.com/radius.PacketServer` instances run concurrently:

| Port | Purpose |
|------|---------|
| 1812 | Authentication |
| 1813 | Accounting |
| 3799 | CoA (optional, controlled by `ENABLE_COA`) |

Secrets are looked up per-NAS by remote address at packet time. The NAS table is
loaded from PostgreSQL and refreshed periodically.

### In-memory state

`radius.Service` keeps three maps protected by a `sync.RWMutex`:

- `nases` — NAS devices keyed by IP address.
- `subscribers` — users keyed by username.
- `sessions` — active/stopped sessions keyed by internal UUID.

A background ticker reloads these maps from the database at
`DB_REFRESH_INTERVAL`. All auth/accounting decisions are map lookups; there are
no per-request database queries on the hot path.

### HTTP management API

`echo` serves management routes grouped under `/api/v1`. All management
endpoints require an internal secret via `Authorization: Bearer <token>` (or the
legacy `X-Internal-Secret` header). Health endpoints (`/health`, `/ready`,
etc.) are public.

### Persistence

PostgreSQL stores:

- `radius_users` — subscribers (including voucher users).
- `radius_nas` — NAS devices and shared secrets.
- `radius_sessions` — session records, including cumulative counters.

Migrations are embedded with the binary and applied at startup. A
`schema_migrations` table records which migrations have already run so each one
is applied only once.

### Authentication

Only PAP is supported. Passwords are stored as bcrypt hashes. Access-Accept
responses include a Message-Authenticator attribute (type 80) computed with
HMAC-MD5 over the response using the NAS secret.

## Consequences

- The server is stateless: a second instance can start cold and reach
  consistency after one refresh cycle. Multiple instances can run behind a UDP
  load balancer for RADIUS and an HTTP load balancer for the API.
- In-memory maps make auth/accounting fast, but configuration changes take up
  to one refresh interval to propagate unless the handler calls an eager
  refresh.
- The single-binary design simplifies deployment but keeps all three concerns
  (RADIUS, HTTP, persistence) in one process.
- PAP-only keeps the auth path simple but limits compatibility with clients
  that require CHAP/MS-CHAPv2.

## Alternatives considered

- **Separate processes for RADIUS and HTTP.** Rejected because it would add
  deployment complexity and require inter-process state sharing.
- **Redis for shared state.** Rejected because PostgreSQL already satisfies
  persistence and the goal was minimal moving parts.
- **CHAP/MS-CHAPv2 support.** Rejected to keep the initial scope small and
  avoid storing plaintext passwords.

## Migration strategy

`migrations/001_initial.sql` creates the base tables (`radius_users`,
`radius_nas`, `radius_sessions`) and indexes. Later migrations extend these
 tables rather than duplicating them.

## Related files

- `migrations/001_initial.sql`
- `internal/app/app.go`
- `internal/radius/service.go`
- `internal/radius/http_router.go`
- `internal/radius/auth.go`
- `internal/radius/accounting.go`
- `internal/database/postgres.go`

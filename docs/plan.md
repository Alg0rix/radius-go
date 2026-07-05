# Plan: radius-go — RADIUS Server with HTTP Management API

## Context

Build production-grade RADIUS server with HTTP management API using `layeh.com/radius`. RADIUS core + HTTP API scope, PostgreSQL storage, Echo HTTP framework, PAP auth only.

## Reference

No external reference project. Design is self-contained; implement clean, idiomatic Go from the requirements below.

## Directory Structure

```
radius-go/
├── cmd/api/main.go                    # Entry point
├── internal/
│   ├── app/
│   │   ├── app.go                     # Bootstrap + echo server + graceful shutdown
│   │   └── router.go                  # Route setup delegation
│   ├── config/
│   │   └── config.go                  # Config struct + env loading + Validate()
│   ├── database/
│   │   └── postgres.go               # pgx pool setup (+ migrations on startup)
│   ├── domain/
│   │   └── domain.go                  # Shared domain types (NAS, Subscriber, Session, etc.)
│   ├── internalsecret/
│   │   └── internalsecret.go          # X-Internal-Secret middleware
│   ├── radius/
│   │   ├── service.go                 # RADIUS server core (auth, accounting, CoA, sessions, HTTP handlers)
│   │   ├── repository.go             # PostgreSQL CRUD for users, NAS, sessions
│   │   └── mikrotik_vsa.go           # MikroTik vendor attributes (vendor 14988)
│   ├── runtime/
│   │   ├── bootstrap.go              # DB pool init, shared Dependencies struct
│   │   ├── logger.go                 # zerolog setup
│   │   ├── http.go                   # OK/Fail/Created JSON envelope helpers
│   │   ├── health.go                 # /health /ready handlers
│   │   └── middleware.go             # UseBaseMiddleware (RequestID + Recover)
│   └── httpapi/
│       └── router.go                 # Full REST route registration
├── migrations/
│   └── 001_initial.sql               # Create tables
├── go.mod
└── go.sum
```

## Database Schema (PostgreSQL — 3 tables)

```sql
CREATE TABLE radius_users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username         TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    full_name        TEXT NOT NULL DEFAULT '',
    email            TEXT NOT NULL DEFAULT '',
    enabled          BOOLEAN NOT NULL DEFAULT true,
    simultaneous_use INTEGER NOT NULL DEFAULT 0,
    session_timeout  INTEGER NOT NULL DEFAULT 0,
    idle_timeout     INTEGER NOT NULL DEFAULT 0,
    framed_ip        TEXT NOT NULL DEFAULT '',
    mikrotik_group   TEXT NOT NULL DEFAULT '',
    rate_limit       TEXT NOT NULL DEFAULT '',
    service_type     TEXT NOT NULL DEFAULT 'framed',  -- 'framed' or 'login'
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE radius_nas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    ip_address  TEXT NOT NULL,
    secret      TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE radius_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       TEXT NOT NULL,
    username         TEXT NOT NULL,
    nas_id           TEXT NOT NULL DEFAULT '',
    nas_ip           TEXT NOT NULL DEFAULT '',
    nas_identifier   TEXT NOT NULL DEFAULT '',
    framed_ip        TEXT NOT NULL DEFAULT '',
    calling_station  TEXT NOT NULL DEFAULT '',
    called_station   TEXT NOT NULL DEFAULT '',
    service_type     TEXT NOT NULL DEFAULT 'framed',
    input_octets     BIGINT NOT NULL DEFAULT 0,
    output_octets    BIGINT NOT NULL DEFAULT 0,
    session_time     BIGINT NOT NULL DEFAULT 0,
    session_status   TEXT NOT NULL DEFAULT 'active',  -- 'active', 'stopped', 'stale'
    mikrotik_group   TEXT NOT NULL DEFAULT '',
    rate_limit       TEXT NOT NULL DEFAULT '',
    start_time       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_update      TIMESTAMPTZ NOT NULL DEFAULT now(),
    stop_time        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_radius_sessions_session_id ON radius_sessions(session_id);
CREATE INDEX idx_radius_sessions_username ON radius_sessions(username);
CREATE INDEX idx_radius_sessions_session_status ON radius_sessions(session_status);
```

## Key Types

```go
// domain/domain.go
type SessionState string
const (
    SessionStateActive SessionState = "active"
    SessionStateStop   SessionState = "stopped"
    SessionStateStale  SessionState = "stale"
)

type ServiceType string
const (
    ServiceTypeFramed ServiceType = "framed"
    ServiceTypeLogin  ServiceType = "login"
)

type RadiusUser struct {
    ID               string
    Username         string
    PasswordHash     string
    FullName         string
    Email            string
    Enabled          bool
    SimultaneousUse  int
    SessionTimeout   int
    IdleTimeout      int
    FramedIP         string
    MikrotikGroup    string
    RateLimit        string
    ServiceType      ServiceType
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type NAS struct {
    ID          string
    Name        string
    IPAddress   string
    Secret      string
    Description string
    Enabled     bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type RadiusSession struct {
    ID              string
    SessionID       string
    Username        string
    NASID           string
    NASIP           string
    NASIdentifier   string
    FramedIP        string
    CallingStation  string
    CalledStation   string
    ServiceType     ServiceType
    InputOctets     int64
    OutputOctets    int64
    SessionTime     int64
    SessionStatus   SessionState
    MikrotikGroup   string
    RateLimit       string
    StartTime       time.Time
    LastUpdate      time.Time
    StopTime        *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

## Key Functions & Signatures

### config/config.go
- `Load(serviceName ...string) Config` — env var loading with godotenv
- `(c Config) Validate() error` — production safety checks

### database/postgres.go
- `NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error)` — pgx v5 connection pool
- `RunMigrations(ctx context.Context, pool *pgxpool.Pool) error` — create tables if not exist

### runtime/bootstrap.go
- `type Dependencies struct { DB *pgxpool.Pool; Logger zerolog.Logger; Config config.Config }`
- `Bootstrap(ctx context.Context, cfg config.Config) (*Dependencies, error)` — init DB, logger

### runtime/logger.go
- `NewLogger(cfg config.Config) zerolog.Logger` — structured logging (console or JSON by env)

### runtime/http.go
- `func OK(c echo.Context, data any) error`
- `func Fail(c echo.Context, status int, code, message string, details any) error`
- `func Created(c echo.Context, data any) error`
- `type Envelope struct { Success bool; Data any; Meta any; Error any }`

### runtime/health.go
- `HealthHandler(deps *Dependencies) echo.HandlerFunc` — DB ping check
- `ReadyHandler(deps *Dependencies) echo.HandlerFunc` — always ready

### runtime/middleware.go
- `UseBaseMiddleware(e *echo.Echo, deps *Dependencies)` — RequestID + Recover

### internalsecret/internalsecret.go
- `Require(secret string) echo.MiddlewareFunc` — checks `X-Internal-Secret` header

### radius/service.go — core (no license/Sentry/CHAP/isolation)

**Types:**
- `ListenerConfig` — HTTPAddr, AuthAddr, AccountingAddr, CoAAddr, SharedSecret, EnableCoA
- `Service` struct — deps, repo, config, startedAt, mu, nases map, subscribers map, sessions map, status, PacketServers, ticker, stopRefresh
- `Status` struct — startedAt, config, NASCount, SubscriberCount, ActiveSessions, counters, health
- `CleanupResult` struct — StaleSessionsCleaned, ActiveSessionsKept, CleanedAt

**Functions (on Service):**
- `Run(cfg config.Config) error` — Bootstrap → NewService → Start → Echo HTTP server
- `NewService(deps, cfg) *Service` — init maps, initial DB refresh
- `(s *Service) Start() error` — 3 PacketServers (auth:1812, acct:1813, coa:3799), tickers
- `(s *Service) Shutdown(ctx)` — stop tickers, shutdown servers
- `(s *Service) handleAuth(w, r)` — PAP auth, NAS matching, simultaneous use, accept with attrs
- `(s *Service) handleAccounting(w, r)` — Start/Stop/Interim, session tracking, async DB persist
- `(s *Service) handleCoA(w, r)` — inbound CoA/Disconnect from NAS
- `(s *Service) handleDisconnectUser(c)` — HTTP POST PoD (disconnect sessions by username)
- `(s *Service) handleCoAChange(c)` — HTTP POST CoA (change user profile on active sessions)
- `(s *Service) handleSessionCleanup(c)` — HTTP POST cleanup stale sessions
- `(s *Service) handleSessionReconcile(c)` — HTTP POST merge DB sessions into memory
- `(s *Service) cleanupStaleSessions(ctx) (CleanupResult, error)` — mark >24h sessions stale
- `(s *Service) DisconnectUser(ctx, username, reason) (int, error)` — PoD to NAS + DB fallback
- `(s *Service) ChangeUserProfile(ctx, username, rateLimit, group) (CoaChangeResult, error)` — CoA to NAS
- `(s *Service) refreshFromDB(ctx) error` — reload NAS + users from DB into memory
- `(s *Service) ListNAS() []domain.NAS`
- `(s *Service) ListSubscribers() []domain.Subscriber`
- `(s *Service) ListSessions() []domain.RadiusSession`
- `(s *Service) Snapshot() Status`
- `radiusSecretSource.RADIUSSecret(ctx, remoteAddr) ([]byte, error)` — per-NAS secret lookup
- `addMessageAuthenticator(packet) error` — HMAC-MD5 Message-Authenticator
- `passwordMatches(hash, supplied string) bool` — bcrypt compare

### radius/repository.go
- `type Repository struct { db *pgxpool.Pool }`
- `(r *Repository) ListUsers(ctx) ([]domain.RadiusUser, error)`
- `(r *Repository) CreateUser(ctx, user) error`
- `(r *Repository) UpdateUser(ctx, user) error`
- `(r *Repository) DeleteUser(ctx, id) error`
- `(r *Repository) ListNAS(ctx) ([]domain.NAS, error)`
- `(r *Repository) CreateNAS(ctx, nas) error`
- `(r *Repository) UpdateNAS(ctx, nas) error`
- `(r *Repository) DeleteNAS(ctx, id) error`
- `(r *Repository) UpsertSession(ctx, session) error`
- `(r *Repository) ListActiveSessions(ctx) ([]domain.RadiusSession, error)`
- `(r *Repository) ListActiveSessionsByUsername(ctx, username) ([]domain.RadiusSession, error)`

### radius/mikrotik_vsa.go
- `MikrotikGroup_SetString(p *radius.Packet, value string) error`
- `MikrotikRateLimit_SetString(p *radius.Packet, value string) error`
- `mikrotikVSAGetString(p *radius.Packet, vendorType byte) string`

### httpapi/router.go
Full CRUD routes:
```
GET  /health, /ready, /healthz, /readyz                              (public)
GET  /api/v1/radius/status                                           (internal-secret)
GET  /api/v1/radius/nases                                            (internal-secret)
POST /api/v1/radius/nases                                            (internal-secret)
PUT  /api/v1/radius/nases/:id                                        (internal-secret)
DELETE /api/v1/radius/nases/:id                                     (internal-secret)
GET  /api/v1/radius/subscribers                                      (internal-secret)
POST /api/v1/radius/subscribers                                      (internal-secret)
PUT  /api/v1/radius/subscribers/:id                                  (internal-secret)
DELETE /api/v1/radius/subscribers/:id                               (internal-secret)
GET  /api/v1/radius/sessions                                         (internal-secret)
POST /api/v1/radius/sessions/disconnect                             (internal-secret)
POST /api/v1/radius/subscribers/coa-change                          (internal-secret)
POST /api/v1/radius/sessions/cleanup                                (internal-secret)
POST /api/v1/radius/sessions/reconcile                              (internal-secret)
```

## Dependency Injection Flow

```
cmd/api/main.go
  → config.Load("radius")
  → app.Run(cfg)
    → runtime.Bootstrap(ctx, cfg) → Dependencies{DB, Logger, Config}
    → radius.NewService(deps, cfg) → *Service
    → svc.Start() → 3 PacketServers + goroutines + tickers
    → echo.New() → middleware → health routes → internal-secret → httpapi routes
    → e.Start(addr) + signal handling + graceful shutdown
```

## Dependencies (go.mod)

```
module github.com/your-org/radius-go
go 1.22+

require (
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.10.0
    github.com/joho/godotenv v1.5.1
    github.com/labstack/echo/v4 v4.14.0
    github.com/rs/zerolog v1.35.1
    golang.org/x/crypto v0.53.0
    layeh.com/radius v0.0.0-20231213012653-1006025d24f8
)
```

## Implementation Order

1. **go.mod + cmd/api/main.go** — module init, entry point
2. **internal/config/config.go** — env loading, validation
3. **internal/database/postgres.go** — pgx pool + migrations
4. **migrations/001_initial.sql** — create tables
5. **internal/domain/domain.go** — shared types
6. **internal/runtime/** — logger, http helpers, health, middleware, bootstrap
7. **internal/internalsecret/internalsecret.go** — X-Internal-Secret middleware
8. **internal/radius/mikrotik_vsa.go** — MikroTik VSA (standalone, no deps)
9. **internal/radius/repository.go** — DB CRUD
10. **internal/radius/service.go** — RADIUS core (auth, accounting, CoA, sessions, disconnect)
11. **internal/httpapi/router.go** — REST route registration
12. **internal/app/app.go + router.go** — bootstrap + echo server lifecycle

## Simplifications

- No license checks (remove `licenseActive()`)
- No Sentry (remove `platformsentry`)
- No Redis (remove Redis dependency)
- No CHAP/MS-CHAPv2 (remove `rfc2759`, `rfc3079`, `microsoft` vendor)
- No PPPoE/Hotspot profiles (single flat user model, `ServiceType` toggle)
- No isolation pool (remove `isolationPool`, `isolationSettings`)
- No `netutil.Loopback` (use simple `fmt.Sprintf("127.0.0.1:%d", port)`)
- No `runtime.NewID()` (use `uuid.New().String()`)
- Single `Config` struct (no separate appconfig alias needed)

## Verification

1. `go build ./cmd/api` — compiles
2. Start PostgreSQL, set `DB_DSN` env, set `INTERNAL_SECRET` env
3. `go run ./cmd/api` — starts, logs health, begins listening on UDP :1812/:1813/:3799 + HTTP :8083
4. `curl http://localhost:8083/health` → `{"success":true,"data":{"status":"ok"}}`
5. `curl -H "X-Internal-Secret: <secret>" http://localhost:8083/api/v1/radius/status` → status JSON
6. `curl -X POST -H "X-Internal-Secret: <secret>" -H "Content-Type: application/json" -d '{"name":"TestNAS","ip_address":"10.0.0.1","secret":"testing123"}' http://localhost:8083/api/v1/radius/nases` → create NAS
7. `curl -X POST -H "X-Internal-Secret: <secret>" -H "Content-Type: application/json" -d '{"username":"testuser","password":"testpass"}' http://localhost:8083/api/v1/radius/subscribers` → create user
8. `echo "User-Name=testuser,User-Password=testpass" | radclient -x 127.0.0.1:1812 auth testing123` → Access-Accept
9. `curl -X POST -H "X-Internal-Secret: <secret>" -H "Content-Type: application/json" -d '{"username":"testuser"}' http://localhost:8083/api/v1/radius/sessions/disconnect` → disconnect

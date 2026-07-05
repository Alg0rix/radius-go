# radius-go

Production-grade RADIUS server with HTTP management API.

Uses [layeh.com/radius](https://layeh.com/radius) for the RADIUS protocol, [labstack/echo](https://echo.labstack.com) for the HTTP API, and PostgreSQL for persistence.

## Quick start

```bash
# Set required env vars
export DB_DSN="postgres://user:pass@localhost:5432/radius?sslmode=disable"
export INTERNAL_SECRET="your-secret-here"

# Run
go run ./cmd/api
```

The server listens on:
- **HTTP** `:8083` — management API + health endpoints
- **UDP** `:1812` — RADIUS authentication
- **UDP** `:1813` — RADIUS accounting
- **UDP** `:3799` — RADIUS CoA (opt-in via `ENABLE_COA=true`)

## Configuration

| Env | Default | Purpose |
|-----|---------|---------|
| `DB_DSN` | _(required)_ | PostgreSQL connection string |
| `INTERNAL_SECRET` | _(required)_ | Header value for management API auth |
| `HTTP_PORT` | 8083 | HTTP listen port |
| `RADIUS_AUTH_PORT` | 1812 | RADIUS auth port |
| `RADIUS_ACCT_PORT` | 1813 | RADIUS accounting port |
| `RADIUS_COA_PORT` | 3799 | RADIUS CoA port |
| `ENABLE_COA` | false | Enable CoA server |
| `LOG_FORMAT` | console | `json` or `console` |
| `DB_REFRESH_INTERVAL` | 60 | DB→memory refresh (seconds) |
| `SESSION_CLEANUP_PERIOD` | 300 | Stale session cleanup (seconds) |

## API

All management endpoints require `X-Internal-Secret` header matching `INTERNAL_SECRET`.

Health endpoints (`/health`, `/ready`, `/healthz`, `/readyz`) are public.

```
GET    /api/v1/radius/status
GET    /api/v1/radius/nases
POST   /api/v1/radius/nases
PUT    /api/v1/radius/nases/:id
DELETE /api/v1/radius/nases/:id
GET    /api/v1/radius/subscribers
POST   /api/v1/radius/subscribers
PUT    /api/v1/radius/subscribers/:id
DELETE /api/v1/radius/subscribers/:id
GET    /api/v1/radius/sessions
POST   /api/v1/radius/sessions/disconnect
POST   /api/v1/radius/subscribers/coa-change
POST   /api/v1/radius/sessions/cleanup
POST   /api/v1/radius/sessions/reconcile
GET    /api/v1/voucher-packages
POST   /api/v1/voucher-packages
PUT    /api/v1/voucher-packages/:id
DELETE /api/v1/voucher-packages/:id
GET    /api/v1/vouchers
POST   /api/v1/vouchers/generate
GET    /api/v1/vouchers/:code/balance
```

Swagger UI: `http://localhost:8083/swagger/index.html`

## Testing

```bash
# Health check
curl http://localhost:8083/health

# Status (requires auth)
curl -H "X-Internal-Secret: <secret>" http://localhost:8083/api/v1/radius/status

# RADIUS auth test (requires radclient)
echo "User-Name=testuser,User-Password=testpass" | radclient -x 127.0.0.1:1812 auth testing123
```
# radius-go

[![Go Version](https://img.shields.io/github/go-mod/go-version/Alg0rix/radius-go)](https://go.dev/)
[![License](https://img.shields.io/github/license/Alg0rix/radius-go)](./LICENSE)

> **Work in progress** — not yet production-ready.

Production-grade RADIUS server with HTTP management API.

Uses [layeh.com/radius](https://layeh.com/radius) for the RADIUS protocol, [labstack/echo](https://echo.labstack.com) for the HTTP API, and PostgreSQL for persistence.

## Table of Contents

- [About The Project](#about-the-project)
  - [Built With](#built-with)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Configuration](#configuration)
- [API](#api)
- [CLI](#cli)
- [Testing](#testing)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## About The Project

`radius-go` is a single-binary RADIUS server that combines UDP-based RADIUS services (authentication, accounting, and optional CoA) with a RESTful HTTP management API. It is designed to be stateless and horizontally scalable: all persistent state lives in PostgreSQL, while in-memory caches are refreshed from the database on a configurable interval.

### Built With

- [![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
- [![Echo](https://img.shields.io/badge/Echo-v4-00ADD8?logo=go&logoColor=white)](https://echo.labstack.com/)
- [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16%2B-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
- [![radius](https://img.shields.io/badge/layeh.com%2Fradius-RADIUS-blue)](https://layeh.com/radius)

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 14+
- `radclient` (optional, for RADIUS testing — usually in the `freeradius-utils` package)

### Installation

1. Clone the repo

   ```bash
   git clone https://github.com/Alg0rix/radius-go.git
   cd radius-go
   ```

2. Copy environment variables

   ```bash
   export DB_DSN="postgres://user:pass@localhost:5432/radius?sslmode=disable"
   export INTERNAL_SECRET="your-secret-here"
   ```

3. Run the server

   ```bash
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
| `STALE_SESSION_TIMEOUT` | 86400 | Age at which a session becomes stale (seconds) |

## API

All management endpoints require `Authorization: Bearer <INTERNAL_SECRET>` (the deprecated `X-Internal-Secret` header is still accepted as a fallback).

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

## CLI

`radiusctl` is a command-line client for the management API. Build and run it from the module root:

```bash
go run ./cmd/radiusctl --help
```

Configure the target with `--server` and `--secret`, or with the `RADIUS_SERVER` and `RADIUS_SECRET` env vars. Add `--json` for machine-readable output.

```bash
export RADIUS_SERVER=http://localhost:8083
export RADIUS_SECRET="$INTERNAL_SECRET"

radiusctl status
radiusctl nas list
radiusctl nas create --name edge --ip 10.0.0.1 --secret sharedkey
radiusctl subscriber create --username alice --password s3cret
radiusctl subscriber list
radiusctl session list
radiusctl session disconnect --username alice
radiusctl session coa-change --username alice --rate-limit 5M/5M
radiusctl voucher package list
radiusctl voucher generate --package-id <uuid> --count 5
radiusctl voucher balance --code <voucher-username>
```

Commands mirror the API:

```
radiusctl status
radiusctl nas {list,create,update,delete}
radiusctl subscriber {list,create,update,delete}
radiusctl session {list,disconnect,coa-change,cleanup,reconcile}
radiusctl voucher {list,generate,balance}
radiusctl voucher package {list,create,update,delete}
```

## Testing

```bash
# Health check
curl http://localhost:8083/health

# Status (requires auth)
curl -H "Authorization: Bearer <secret>" http://localhost:8083/api/v1/radius/status

# RADIUS auth test (requires radclient)
echo "User-Name=testuser,User-Password=testpass" | radclient -x 127.0.0.1:1812 auth testing123
```

## Roadmap

- [ ] Add unit and integration test suite
- [ ] Add metrics / Prometheus exporter
- [ ] Add structured request logging middleware
- [ ] Add multi-region HA documentation

See the [open issues](https://github.com/Alg0rix/radius-go/issues) for more proposed features and known issues.

## Contributing

Contributions are welcome. If you have a suggestion, please fork the repo and open a pull request, or open an issue with the tag `enhancement`.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Acknowledgments

- [layeh.com/radius](https://layeh.com/radius) — RADIUS protocol library
- [labstack/echo](https://echo.labstack.com/) — HTTP web framework
- [jackc/pgx](https://github.com/jackc/pgx) — PostgreSQL driver
- [Best-README-Template](https://github.com/othneildrew/Best-README-Template) — README structure inspiration

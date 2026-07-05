# ADR-002: Voucher System

## Status

Accepted

## Context

The RADIUS server needed a way to sell or distribute time- and/or data-limited
access credentials (vouchers) without creating a separate authentication system.
Vouchers are commonly used for guest Wi-Fi hotspots: a user gets a code, enters
it as both username and password, and is granted access until their time or data
budget is exhausted.

Key requirements:

- Define reusable voucher packages (price, speed, data cap, time limit, max
  concurrent users).
- Generate one or many voucher codes from a package.
- Authenticate vouchers using the same RADIUS path as regular subscribers.
- Track usage (time and bytes) across sessions.
- Enforce limits and disable the voucher automatically when exhausted.
- Support both usage-based time limits (tracked session time) and calendar
  expiry (fixed window from first login).

## Decision

Represent each voucher as a regular `radius_users` row with `is_voucher = true`.
Voucher policy is stored on the package (`voucher_packages`) and copied onto the
subscriber at generation time. This lets vouchers reuse the existing auth,
accounting, session, and subscriber infrastructure.

Usage counters (`usage_seconds_used`, `data_bytes_used`) live on the subscriber
row and are updated incrementally from accounting interim-update and stop
packets. When a limit is hit, the voucher is disabled and an active session is
forcibly disconnected via PoD.

### Voucher package as hotspot profile

A voucher package is also the hotspot profile. It carries `address_pool`,
`primary_dns`, and `secondary_dns` so that hotspot vouchers can emit
`Framed-Pool` and DNS VSAs during Access-Accept. This avoids inventing a second
"hotspot profile" entity.

### Time limit types

- `usage` — counts cumulative `Acct-Session-Time` reported in accounting.
- `calendar` — records `first_login_at` and computes `expires_at` from the
  package's `time_limit_seconds`.

### Generation options

Vouchers can be generated with random or custom codes, and passwords can be:

- `same_as_user` — password equals the username/code (default for printed cards).
- `random` — a separate random password.
- `custom` — an operator-supplied password.

## Consequences

- Vouchers inherit subscriber capabilities (NAS lookup, simultaneous-use limits,
  session tracking) with no duplicate code.
- Disabling a voucher is the same operation as disabling a subscriber.
- Package changes do not affect already-generated vouchers because policy is
  copied at generation time. This is intentional: a sold voucher should not
  change terms after issuance.
- Calendar-expiry vouchers require a first-login marker (`first_login_at`), so
  the expiry clock starts on first successful authentication.
- Data usage must support 64-bit totals, so `Acct-Input-Gigawords` and
  `Acct-Output-Gigawords` are combined with the 32-bit octet counters.
- Voucher data caps are surfaced to NAS devices via `pfSense-Max-Total-Octets`
  and `MikroTik-Total-Limit` VSAs.

## Alternatives considered

- **Separate `vouchers` table.** Rejected because it would duplicate auth
  lookup, session handling, and disable logic already present for subscribers.
- **Reference package at auth time instead of copying fields.** Rejected because
  it would allow package updates to retroactively change sold vouchers, which is
  surprising for prepaid access.
- **Track voucher usage only in sessions and sum on read.** Rejected because it
  requires scanning all historical sessions for every auth decision and makes
  limit enforcement expensive.

## Migration strategy

`migrations/002_vouchers.sql` adds the `voucher_packages` table and the voucher
columns on `radius_users`. Because voucher users are ordinary subscribers, no
auth or accounting schema changes are needed beyond these columns.

## Related files

- `migrations/002_vouchers.sql`
- `internal/domain/domain.go`
- `internal/radius/voucher_service.go`
- `internal/radius/auth.go`
- `internal/radius/accounting.go`
- `internal/radius/handlers.go` (voucher HTTP handlers)
- `internal/radiusctl/voucher.go`

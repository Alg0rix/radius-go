# ADR-001: PPPoE Profiles and Hotspot Profile Extensions

## Status

Accepted

## Context

The RADIUS server had to support two distinct subscriber types:

1. **PPPoE users** — need PPP-layer attributes (Framed-Protocol=PPP, Framed-Pool,
   MTU/MRU, compression, DNS) plus per-session policy (rate-limit, bandwidth,
   session-timeout, idle-timeout, max-total-octets).
2. **Hotspot/voucher users** — managed through voucher packages and need
   address-pool and DNS overrides in addition to existing speed/data-cap rules.

The codebase already supported voucher packages. Adding PPPoE required a
profile-like entity that could be attached to a subscriber without duplicating
the same fields on every user. We also needed a clear precedence rule for
attributes that can be set on both the user and the profile.

## Decision

Introduce a `pppoe_profiles` table/entity and extend `voucher_packages` to serve
as the hotspot profile. Attach a PPPoE profile to a subscriber via
`radius_users.pppoe_profile_id`. Make `voucher_package_id` and
`pppoe_profile_id` mutually exclusive on a single user.

Precedence follows FreeRADIUS semantics: per-user RADIUS attributes
(`radreply`) override profile defaults (`radgroupreply`). Empty per-user fields
fall through to the attached profile.

## Consequences

- **PPPoE auth path** now emits Framed-Protocol, Framed-Pool, Framed-IP-Netmask,
  Framed-MTU, Framed-Compression, Session-Timeout, Idle-Timeout, MikroTik
  Rate-Limit/Address-Pool/Total-Limit, Microsoft DNS VSAs, and pfSense DNS VSAs
  when the attached profile provides them.
- **Hotspot voucher auth path** now emits Framed-Pool and DNS VSAs when the
  voucher package provides them.
- The `effective*` helpers in `internal/radius/auth.go` centralize precedence
  logic and make the rule explicit and testable.
- Keeping hotspot profiles in `voucher_packages` avoids a new entity and matches
  the existing model where a package *is* the hotspot policy.
- Mutual exclusion prevents a subscriber from being both a voucher user and a
  PPPoE user at the same time, which would create conflicting attribute sets.

## Alternatives considered

- **Single profile table for both PPPoE and hotspot.** Rejected because the two
  profiles have different attribute sets and lifecycle owners; merging them
  would create many nullable columns and confuse the domain model.
- **Copy all profile fields onto `radius_users`.** Rejected because it makes
  policy updates painful (every attached user must be updated) and bloats the
  subscriber table.

## Migration strategy

`migrations/003_profiles.sql` creates `pppoe_profiles`, extends
`voucher_packages`, adds `radius_users.pppoe_profile_id`, and the required
indexes. The runner records applied migrations in `schema_migrations` so each
migration runs only once, and the SQL uses `IF NOT EXISTS` for idempotency.

## Related files

- `migrations/003_profiles.sql`
- `internal/domain/domain.go`
- `internal/radius/auth.go`
- `internal/radius/pppoe_attributes.go`
- `internal/radius/pppoe_profile_service.go`
- `internal/radius/pppoe_handlers.go`
- `internal/radius/vsa.go`
- `internal/radius/validate.go`
- `internal/radius/subscriber_handlers.go`

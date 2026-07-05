# Review: PPPoE profiles + voucher/Hotspot profile extensions

## Context
This uncommitted change adds a **PPPoE profiles** feature: a new `pppoe_profiles` table + entity, a new `PPPoEProfileService`, PPP-layer RADIUS attribute emission in the auth path, "effective" precedence helpers (per-user overrides profile), mutual-exclusion validation against voucher packages, a hotspot profile extension to `voucher_packages` (address pool + DNS), Microsoft/pfSense DNS VSAs, MikroTik address-pool/total-limit VSAs, and 5 new HTTP CRUD endpoints. It also updates `AGENTS.md`/`docs/plan.md`.

**Diff size:** ~669 lines across 13 modified + 4 new files. That's at the upper edge of "acceptable if it's a single logical change" — and it bundles schema, domain, repo, auth, validation, handlers, VSA, and docs. Per the sizing guidance this would read more cleanly split horizontally (schema+domain+repo → service/precedence → HTTP handlers).

## Verdict
**Request changes** — two Critical blockers (broken build, non-idempotent migration) plus a real correctness bug in the precedence logic and required hygiene items (file size, error handling, tests, dead code).

---

## Critical: — blocks merge

**Critical: The build is broken.** `go build ./...` and `go vet ./...` both fail:
```
internal/radius/pppoe_attributes.go:26: undefined: rfc2865.FramedPool_Set
internal/radius/pppoe_attributes.go:26: undefined: rfc2865.FramedPool
internal/radius/pppoe_attributes.go:48: undefined: rfc2865.FramedCompression_Value_StacLZSCompression
```
I verified against the vendored `layeh.com/radius/rfc2865/generated.go`:
- The compression constant is **`FramedCompression_Value_StacLZS`** (value 3), not `…StacLZSCompression`.
- `FramedPool` / `FramedPool_Set` **do not exist** in `rfc2865` at all. Framed-Pool is attribute type **88** (RFC 2869), which `layeh.com/radius` doesn't generate a helper for.

This means the entire `radius` package — and therefore `cmd/api` and `cmd/radiusctl` — does not compile. Nothing can be merged or run. Remedy: `FramedCompression_AddStac` → use `rfc2865.FramedCompression_Value_StacLZS`; `FramedPool_SetString` → set the attribute directly via its type code, e.g. `p.Set(88, radius.NewString(pool))` (define a named const `framedPoolType radius.Type = 88`).

**Critical: Migration `003_profiles.sql` is not idempotent and will break every restart after the first.** `internal/database/postgres.go:RunMigrations` re-executes **every** embedded `.sql` file on **every** startup — there is no migration-version table; it just `pool.Exec(ctx, string(sql))` for each sorted file. Migrations `001` and `002` are deliberately idempotent (`CREATE TABLE IF NOT EXISTS`, `ALTER TABLE … ADD COLUMN IF NOT EXISTS`) — `002` even carries the comment *"All DDL uses IF NOT EXISTS so re-runs are safe."* Migration `003` drops that convention entirely:
```sql
CREATE TABLE pppoe_profiles (...)           -- fails: relation already exists
CREATE INDEX idx_pppoe_profiles_enabled ... -- fails: relation already exists
ALTER TABLE voucher_packages ADD COLUMN ... -- fails: column already exists
ALTER TABLE radius_users ADD COLUMN ...     -- fails: column already exists
CREATE INDEX idx_radius_users_pppoe_profile_id -- fails: relation already exists
```
On the **second boot** (or HA side-by-side instance start, which AGENTS.md assumes), `RunMigrations` returns an error and the server fails to start. Remedy: make `003` idempotent to match `001`/`002` — `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE … ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`.

---

## Required (must address before merge)

`effectiveRateLimit` violates the documented precedence. `auth.go`:
```go
func effectiveRateLimit(user domain.RadiusUser) string {
    if user.RateLimit != "" { return user.RateLimit }                 // per-user string
    if user.PPPoEProfile != nil {
        if user.PPPoEProfile.RateLimit != "" { return … }            // profile string
        if rl := formatBandwidthKbps(profile.Bandwidth…); rl != "" { return rl } // profile bandwidth
    }
    return formatRateLimit(user)                                      // per-user bandwidth (LAST)
}
```
AGENTS.md states *"per-user RADIUS attributes override profile defaults when set; empty per-user fields fall through to the attached profile."* But here **profile bandwidth wins over per-user bandwidth** — a user with `BandwidthMaxUp/Down` set (and no `RateLimit` string) attached to a profile with numeric bandwidth gets the **profile's** limit, ignoring their own. This is also inconsistent with `effectiveBandwidth`, which correctly does per-user-first:
```go
up, down := user.BandwidthMaxUp, user.BandwidthMaxDown   // per-user first
if up == 0  { up = profile.BandwidthMaxUp }              // profile only when per-user empty
```
Concrete failure: user `BandwidthMaxUp=2000`, profile `BandwidthMaxUp=512` → emitted rate limit is `512K/…`, not `2000K/…`. Remedy: check per-user bandwidth (`formatRateLimit(user)`) **before** profile bandwidth, so the order is per-user string → per-user bandwidth → profile string → profile bandwidth.

`internal/radius/handlers.go` is **537 lines** — over the AGENTS.md hard rule *"Every source file < 500 LOC."* This change added ~31 lines (PPPoE profile-ID handling in create/update subscriber) to a file that was already at the limit. Per the review guidance: *decompose, then add*. Remedy: extract the subscriber create/update handlers (or the PPPoE handler group) into their own file(s) in the `radius` package before/alongside this change.

`HandleGetPPPoEProfile` and `HandleUpdatePPPoEProfile` swallow **all** errors as 404 and pass `nil` as the underlying error, dropping real DB failures from logging and returning 404 on a DB outage. `pppoe_handlers.go`:
```go
profile, err := s.pppoe.GetProfile(ctx, id)
if err != nil {
    return s.fail(c, http.StatusNotFound, "not_found", "pppoe profile not found", nil) // err discarded
}
```
`GetProfile` returns `pgx.ErrNoRows` (→ "not found") **and** genuine DB errors (returned unwrapped). Both become a 404 with no log. Compare `HandleCreatePPPoEProfile`/`HandleDeletePPPoEProfile`, which correctly return 500 and pass `err`. Remedy: branch on `errors.Is(err, pgx.ErrNoRows)` → 404; else → 500 and pass `err` to `s.fail` so it's logged. (Same fix in `HandleUpdatePPPoEProfile`.)

No tests exist for this change — in fact `find . -name '*_test.go'` returns **zero** files in the entire repo. This change introduces pure, trivially-testable business logic that *encodes the spec*: `effectiveRateLimit`, `effectiveBandwidth`, `effectiveMaxTotalOctets`, `effectiveSessionTimeout`, `effectiveIdleTimeout` (the "radreply over radgroupreply" precedence), plus `validateMutualExclusion`, `validatePPPoEProfile`, `validateUpdatePPPoEProfile`, `validIPv4`, `formatBandwidthKbps`. The precedence logic is exactly the kind of thing that silently regresses without tests. Remedy: add a table-driven `auth_test.go` covering per-user-override / profile-fallthrough / mutual-exclusion / IPv4 validation edge cases. (Running `go test ./...` is part of the verification gate — right now there's nothing to run.)

Dead code introduced by this change: `MikrotikRecvLimit_Set` and `MikrotikXmitLimit_Set` (and their constants `mikrotikRecvLimitType`/`mikrotikXmitLimitType`) in `mikrotik_vsa.go` are defined but **never called** anywhere (only `MikrotikTotalLimit_Set` and `MikrotikAddressPool_SetString` are wired into `auth.go`). Per dead-code hygiene: should these unused helpers be removed? If recv/xmit limits are planned, file that intent separately; otherwise delete them now rather than leaving speculative API surface in an application binary.

---

## Optional / Consider

`HandleCreatePPPoEProfile` does **not** call `s.refreshFromDBAsync()`, while `HandleUpdatePPPoEProfile` and `HandleDeletePPPoEProfile` do. A freshly created profile is therefore absent from the in-memory `profileMap` until the next ticker, so a subscriber assigned + authenticated immediately after creation won't receive the profile's PPP attributes until the next refresh. Consider calling `refreshFromDBAsync()` on create for consistency with update/delete (and with the eager-refresh pattern used by the voucher/NAS handlers).

There are now **three near-identical** `*VSASetUint32` helpers — `mikrotikVSASetUint32` (mikrotik_vsa.go), `microsoftVSASetUint32` (pppoe_attributes.go), and `pfSenseVSASetUint32` (pfsense_vsa.go). Each builds the same 6-byte sub-attribute (`[type][len=6][4-byte BE value]`), wraps it in `radius.NewVendorSpecific(vendorID, sub)`, and adds it. We're at the third use case — the guideline threshold for generalization. Consider one shared `vsaSetUint32(p *radius.Packet, vendorID uint32, vendorType byte, value uint32) error` that the three vendor wrappers delegate to.

`effectiveMaxTotalOctets` uses the magic literal `4294967295` and silently truncates `int64` profile caps (`BIGINT` column) to 32 bits. Use `math.MaxUint32` for clarity, and add a one-line comment noting the 4 GiB RADIUS-attribute ceiling (Acct-Input/Output-Gigawords would be needed to exceed it — deliberately out of scope).


---

## FYI — no action needed

- `refreshFromDB` filters **disabled** PPPoE profiles out of the in-memory map but keeps **disabled** voucher packages. This looks intentional (a disabled profile → no PPP attrs on next refresh; `ValidateAssignment` already rejects assigning to a disabled profile), but the asymmetry is non-obvious — a comment would help future readers.
- Pre-existing latent duplicate-`Session-Timeout` possibility: the per-user/profile `Session-Timeout` (now via `effectiveSessionTimeout`) and the voucher near-expiry `Session-Timeout` both use `_Add` (append). For normal PPPoE users (`IsVoucher=false`) and normal voucher users (no profile) there's no duplicate; it only arises for a misconfigured user that is both `IsVoucher` and has a profile/time-limit fields — which mutual exclusion on `voucher_package_id` vs `pppoe_profile_id` doesn't fully prevent (it doesn't check `IsVoucher`). Not introduced by this change; flagging because `effectiveSessionTimeout` now shares the emit path. A future `_Set`/guard would make it RFC-compliant (Session-Timeout should appear at most once).
- `emitDNS` emits the same DNS in **both** Microsoft (311) and pfSense VSA formats in one Access-Accept. Reasonable multi-NAS-vendor choice; just be aware some NAS may surface both.
- `gofmt -l` reports nearly every file in the repo as dirty (pre-existing, repo-wide — not introduced here). Once the build is fixed, a one-off `gofmt -w internal/ cmd/` would clean it up.

## Nit
- `mikrotikVSASetUint32` carries a comment about "big-endian for int helpers but we are writing raw bytes" that's slightly muddy; the sub-attribute format `[type][len][value]` with `len=6` is correct, the comment could just say that.

---

## Review checklist

### Context
- [x] I understand what this change does and why (PPPoE profiles + Hotspot profile extension + new VSAs + precedence model)

### Correctness
- [ ] Change matches spec — **precedence bug in `effectiveRateLimit`** (R1)
- [ ] Edge cases handled — magic-number truncation (O3); disabled-profile filtering intentional (F1)
- [ ] Error paths handled — **Get/Update swallow DB errors as 404** (R3)
- [ ] Tests cover the change — **no tests at all** (R4)

### Readability
- [x] Names are clear and consistent (`effective*`, `validateMutualExclusion`, `emitDNS`)
- [x] Logic is straightforward; the `effective*` helpers are a clean factoring of the precedence model
- [ ] Dead code: `MikrotikRecvLimit_Set`/`MikrotikXmitLimit_Set` unused (R5)

### Architecture
- [x] Follows existing patterns (repository scan helpers, service+handler split, VSA files per vendor)
- [x] No new dependencies; uses existing `layeh.com/radius`, `pgx`, `echo`
- [ ] Appropriate file size — **handlers.go 537 LOC > 500 hard limit** (R2)
- [x] Refactor reduces complexity — the `effective*` extraction is a genuine simplification of the auth path (good)

### Security
- [x] SQL is parameterized everywhere (`$1…` placeholders; no string concatenation)
- [x] Input validated at boundaries (`validatePPPoEProfile`, `validIPv4`, `validUUID`, `limitString`)
- [x] Endpoints behind `internalsecret.Require` via the route group (http_router.go)
- [x] No secrets in code; bcrypt retained for passwords
- [x] External RADIUS/user data treated as untrusted (ParseIP guards, To4 checks)

### Performance
- [x] No N+1 — profiles + packages loaded once per refresh into maps, joined in-memory
- [x] `ListProfiles`/`ListPackages` are bounded single queries (no pagination, but profile/package counts are small and bounded by admin action — acceptable)
- [x] Auth hot path does map lookups only; no per-request DB hits

### Verification
- [ ] Tests pass — **none exist** (R4)
- [ ] Build succeeds — **`go build ./...` FAILS** (C1)
- [ ] Manual verification — not possible until build is fixed; migration idempotency unverified at runtime (C2)

---

## Verification status (my own checks)
- `go build ./...` → **fails** (3 undefined symbols in `pppoe_attributes.go`)
- `go vet ./...` → **fails** (same symbols)
- `find . -name '*_test.go'` → **empty** (no tests in repo)
- `wc -l internal/radius/handlers.go` → **537** (> 500 LOC hard rule)
- `layeh.com/radius/rfc2865/generated.go` grep → confirms `FramedCompression_Value_StacLZS` (not `…Compression`) and absence of `FramedPool`/`FramedPool_Set`/attr 88
- `internal/database/postgres.go:RunMigrations` → re-executes all SQL each startup, no version tracking; `001`/`002` idempotent, `003` is not

## Summary
The design is sound and the `effective*` precedence extraction is a genuinely good simplification, but this cannot merge as-is. The two Criticals are hard blockers — **the code doesn't compile**, and **the migration will brick the server on the second startup** (directly undermining the HA-ready assumption in AGENTS.md). Beyond those, the `effectiveRateLimit` precedence inversion is a real correctness bug that contradicts the documented per-user-overrides-profile rule, `handlers.go` breaches the <500 LOC hard rule, the Get/Update handlers hide DB errors as 404s, and the new precedence logic ships with zero tests. I'd also remove the two unused MikroTik helpers. Once C1/C2 and R1–R5 are addressed, this is an approve.


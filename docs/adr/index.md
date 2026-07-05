# Architecture Decision Records

This directory records significant architectural decisions for radius-go.

| ADR | Title | Description |
|-----|-------|-------------|
| [ADR-001](001-base-architecture.md) | Base RADIUS Server Architecture | Single-binary design: UDP RADIUS core, HTTP management API, PostgreSQL persistence, and in-memory caches. |
| [ADR-002](002-voucher-system.md) | Voucher System | Time- and data-limited hotspot vouchers modeled as regular subscribers with voucher-specific usage tracking. |
| [ADR-003](003-pppoe-profiles.md) | PPPoE Profiles and Hotspot Profile Extensions | Reusable PPPoE policy bundles and extending voucher packages to serve as hotspot profiles. |

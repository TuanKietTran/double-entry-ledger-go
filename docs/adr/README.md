# Architecture decisions

The platform ADR filenames are shared across the five synthetic services; service-specific decisions remain listed below.

| Shared decision | Ledger application |
|---|---|
| [Platform 01: CQRS](platform-01-cqrs.md) | Commands and queries use separate handlers and repository ports. |
| [Platform 02: Exact money](platform-02-exact-money.md) | Exact decimal amounts, explicit currencies, balanced single-currency journals. |
| [Platform 03: Idempotency](platform-03-idempotency.md) | Keys bind to payloads; replays do not duplicate journal effects. |
| [Platform 04: Outbox and resilience](platform-04-outbox-and-resilience.md) | Outbox/downstream retries are not applicable: ledger makes no downstream calls. Playback retries transient HTTP faults to exercise idempotent writes. |
| [Platform 05: Persistence and testing](platform-05-persistence-and-testing.md) | Memory and PostgreSQL adapters share a repository contract; tests include embedded PostgreSQL. |

Service-specific ADRs: [exact money and journal invariants](0001-exact-money-and-journal-invariants.md), [CQRS and persistence](0002-cqrs-and-persistence.md), and [idempotency, reversal, and versions](0003-idempotency-reversal-and-versions.md).

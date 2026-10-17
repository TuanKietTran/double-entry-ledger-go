# Definition of done (checkpoint 2026-10-18)

A **Met** row names executable evidence. This checkpoint measures repository/test evidence only; no production deployment or live-process performance claim is made.

## Met

| Requirement | Evidence |
|---|---|
| Exact positive balanced single-currency journals, strict directions, explicit accounts | `go test -p 2 ./... -race -count=1 -cover`; domain package coverage 93.7%; account/domain/repository invariant tests |
| Payload-bound idempotent posting and append-only full reversal | command/API and in-memory repository tests; `TestPostgresLedger` (embedded Postgres, 7 subtests) PASS natively on Windows, 28.03s, 2026-10-18 |
| Account versions and query interfaces (lookup, entries, as-of balance, trial balance) | API/query/repository tests in full race suite |
| Memory repository implementation | repository tests exercise the in-memory adapter in the full race suite |
| HTTP routes and statuses contract parsed and checked against registrations | `TestOpenAPIListsCurrentRoutesAndStatuses` parses YAML and checks exact route set and required statuses |
| Operational endpoints and bounded server configuration | `cmd/server` tests; full race suite |
| Domain >=90%, weighted non-cmd/internal >=80% | domain 93.7%; weighted internal 84.0%, measured in card 5d with `-coverpkg=./internal/...` |

## Partial

| Requirement | Limitation |
|---|---|
| Overall runtime resilience | Health/readiness/metrics, request IDs, body limits and graceful shutdown exist; no downstream clients/outbox/retry circuit breaker are needed or implemented because ledger consumes no service. Real-process smoke and shutdown probe were not run. |
| Contract validation | YAML syntax, route set and required statuses are tested; all OpenAPI schema semantics and every error/result body shape are not exhaustively validated. |
| PostgreSQL parity/concurrency | Partial: `TestPostgresLedger` passes natively (7 subtests, includes idempotency and optimistic-version atomicity). It skips only when run as root. Simultaneous multi-connection conflicts are not covered. |
| Operations portability | CI workflow is authored but not run; Docker unavailable; Linux/macOS unverified. |

## Unmet / not claimed

| Requirement | Limitation |
|---|---|
| Stable cursor pagination under concurrent appends | Current deterministic offset pagination may shift as new entries arrive. |
| Production durability, real money, external integrations, deployment | Out of scope; no such claims are made. |

Playback is regression evidence only, not real-process load evidence. The legacy storyboard predates payload-bound replay.

# Definition of done (checkpoint 2026-10-18)

A **Met** row names executable evidence. This checkpoint measures repository/test evidence only; no production deployment or live-process performance claim is made.

## Met

| Requirement | Evidence |
|---|---|
| Exact positive balanced single-currency journals, strict directions, explicit accounts | `go test -p 2 ./... -race -count=1 -cover`; domain package coverage 93.7%; account/domain/repository invariant tests |
| Payload-bound idempotent posting and append-only full reversal | command/API and in-memory repository tests; `TestPostgresLedger` (embedded Postgres, 7 subtests) PASS natively on Windows, 28.03s, 2026-10-18 |
| Account versions and query interfaces (lookup, entries, as-of balance, trial balance) | API/query/repository tests in full race suite |
| Memory repository implementation | repository tests exercise the in-memory adapter in the full race suite |
| HTTP routes and statuses contract parsed and checked against registrations | `TestOpenAPIListsCurrentRoutesAndStatuses` parses YAML and checks exact route set and required statuses; `TestOpenAPIResponseProperties` and `TestHandlerBodiesMatchOpenAPIResponses` check success/error schemas and representative bodies |
| Operational endpoints and bounded server configuration | `cmd/server` tests; full race suite |
| Domain >=90%, weighted non-cmd/internal >=80% | domain 93.7%; weighted internal 87.4%, measured 2026-10-18 on Go 1.27.1 macOS arm64 with `-coverpkg=./internal/...` |

## Partial

| Requirement | Limitation |
|---|---|
| Overall runtime resilience | Health/readiness/metrics, request IDs, body limits and graceful shutdown exist; no downstream clients/outbox/retry circuit breaker are needed or implemented because ledger consumes no service. Real-process smoke and shutdown probe were not run. |
| Contract validation | All operations declare success/error schemas. Property sets, required fields and representative runtime bodies are tested for types/enums/references, including 400/404/409 errors. Full OpenAPI semantics, formats/bounds and injected 500 bodies are not exhaustively validated. |
| PostgreSQL parity/concurrency | Partial: `TestPostgresLedger` passes natively on Windows and macOS arm64 (7 subtests, includes idempotency and optimistic-version atomicity). It skips only when run as root. Simultaneous multi-connection conflicts are not covered. |
| Operations portability | macOS arm64 build/vet/race tests passed on Go 1.27.1 (2026-10-18); CI workflow is authored but not run; Docker unavailable; Linux unverified. |

## Unmet / not claimed

| Requirement | Limitation |
|---|---|
| Stable cursor pagination under concurrent appends | Current deterministic offset pagination may shift as new entries arrive. |
| Production durability, real money, external integrations, deployment | Out of scope; no such claims are made. |

Playback is regression evidence only, not real-process load evidence. The legacy storyboard predates payload-bound replay.

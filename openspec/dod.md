# Definition of done (checkpoint 2026-10-18)

A **Met** row names executable evidence. This checkpoint measures repository/test evidence only; no production deployment or live-process performance claim is made.

## Met

| Requirement | Evidence |
|---|---|
| Exact positive balanced single-currency journals, strict directions, explicit accounts | `go test -p 2 ./... -race -count=1 -cover`; domain package coverage 93.7%; account/domain/repository invariant tests |
| Payload-bound idempotent posting and append-only full reversal | command/API and in-memory repository tests; `TestPostgresLedger` (embedded Postgres, 7 subtests) PASS natively on Windows, 28.03s, 2026-10-18 |
| Account versions and query interfaces (lookup, entries, as-of balance, trial balance) | API/query/repository tests in full race suite |
| Memory repository implementation | repository tests exercise the in-memory adapter in the full race suite |
| Independent PostgreSQL connection conflicts and tied pagination | `TestPostgresLedger/independent_connection_conflicts` (replay, payload/version conflict, unique reversal) and `TestPostgresLedger/timestamp_tie_pagination`; macOS race run 2026-10-18, PASS 49.005s for the complete specific suite |
| HTTP routes and statuses contract parsed and checked against registrations | `TestOpenAPIListsCurrentRoutesAndStatuses` parses YAML and checks exact route set and required statuses; `TestOpenAPIResponseProperties` and `TestHandlerBodiesMatchOpenAPIResponses` check success/error schemas and representative bodies |
| Operational endpoints and bounded server configuration | `cmd/server` tests; full race suite |
| Domain >=90%, weighted non-cmd/internal >=80% | domain 93.7%; weighted internal 87.6%, measured 2026-10-18 on Go 1.27.1 macOS arm64 with `-coverpkg=./internal/...` |

## Partial

| Requirement | Limitation |
|---|---|
| Overall runtime resilience | Health/readiness/metrics, request IDs, body limits and graceful shutdown exist; no downstream clients/outbox/retry circuit breaker are needed or implemented because ledger consumes no service. A native real-process smoke ((local gate log, not committed), built binary, in-memory repository, macOS arm64, 2026-10-18) passed: healthz/readyz 200, account creation, post, exact replay, changed-payload 409, balance 12.34, request-ID echo and SIGTERM exit 0 (log (local gate log, not committed)). A PostgreSQL-mode real-process smoke ((local gate log, not committed), cached embedded PostgreSQL 16.9 on 127.0.0.1:55433, throwaway data dir, both migrations applied externally, built binary, macOS arm64, 2026-10-19) also passed: readiness, post, exact replay, changed-payload 409, balance 12.34, transaction lookup with one DEBIT and one CREDIT entry of 12.34, restart persistence (balance, lookup, replay, 409 unchanged) and SIGTERM exit 0 twice (log (local gate log, not committed)). Neither smoke covers load, Linux/CI or remote databases. Panics return the JSON error envelope via `api.Recoverer`. |
| Contract validation | All operations declare success/error schemas. Property sets, required fields and representative runtime bodies are tested for types/enums/references, including 400/404/409 errors. Full OpenAPI semantics, formats/bounds and injected 500 bodies are not exhaustively validated. |
| PostgreSQL parity/concurrency | `TestPostgresLedger` passes natively on Windows (original 7 subtests) and macOS arm64 (now 9 top-level subtests and 4 nested concurrent cases). Independent connection replay/conflicts/reversals and timestamp ties are covered. It skips only as root. Network interruption, process-crash recovery and exhaustive database fault behavior remain unverified. |
| Operations portability | macOS arm64 build/vet/race tests passed on Go 1.27.1 (2026-10-18); CI workflow is authored but not run; Docker unavailable; Linux unverified. |

## Unmet / not claimed

| Requirement | Limitation |
|---|---|
| Stable cursor pagination under concurrent appends | Current deterministic offset pagination may shift as new entries arrive. |
| Production durability, real money, external integrations, deployment | Out of scope; no such claims are made. |

Playback is regression evidence only, not real-process load evidence. The legacy storyboard predates payload-bound replay.

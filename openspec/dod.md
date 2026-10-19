# Definition of done (evidence checkpoint 2026-10-19)

This status covers repository tests and the in-memory HTTP runtime only. No production deployment, hosted CI, remote database reliability, or sustained throughput claim is made.

## Met

| Requirement | Evidence |
|---|---|
| Exact positive balanced single-currency journals, strict directions, explicit accounts | `go test -p 2 ./... -race -count=1 -cover`; domain coverage 93.7%; domain and repository invariant tests. |
| Payload-bound idempotent posting, append-only reversal, and optimistic versions | command/API and in-memory repository tests; `TestPostgresLedger` covers replay/conflict and reversal concurrency. |
| CQRS query routes for account, transaction, entries, balance/as-of and trial balance | API/query/repository tests in the full race suite. |
| Stable cursor entry paging plus legacy offset compatibility | `TestCursorPaginationContinuesAcrossConcurrentAppend`, `testPostgresTimestampTiePagination`, and OpenAPI route/schema tests. |
| R1 loopback HTTP replay, balanced state, duplicate suppression, and transient-failure retries | `TestR1TenThousandProductionHTTPRequests`; 10,000 logical submissions, 200 duplicate bodies/keys, 9,800 unique effects, 503/429/timeout injection, terminal success, versions and trial balance asserted. |
| Operational endpoints, bounded request bodies, request IDs and graceful shutdown behavior | API and `cmd/server` tests; full race suite. |
| Domain and weighted internal coverage targets | Domain 93.7%; weighted `./internal/...` statement coverage 87.7% using `go test -p 2 ./... -race -count=1 -coverpkg=./internal/... -coverprofile=<profile>` and `go tool cover -func=<profile>` (Go 1.27.2, Windows/amd64, 2026-10-19). |

## Partial

| Requirement | Limitation |
|---|---|
| PostgreSQL reliability | Embedded PostgreSQL tests cover independent-connection conflicts, replay, reversals and tied/keyset pagination. Network interruption, crash recovery and exhaustive database fault behavior are unverified. |
| OpenAPI validation | Route/status registration, response properties and representative handler bodies are tested; exhaustive OpenAPI semantic/format validation is not. |
| CI and portability | CI workflow is authored but no hosted run is verified. Linux and Docker builds are unverified; Docker is not required for tests. |
| Runtime operations | Health/readiness/metrics, request IDs, body limits, panic recovery and graceful shutdown are implemented. No external downstream calls exist, so a downstream retry, circuit breaker, or outbox is not applicable. |

## Unmet / not claimed

| Requirement | Limitation |
|---|---|
| Production durability and deployment | Memory mode is volatile; PostgreSQL exists but is not represented as an operated production system. |
| Sustained live-process throughput / platform soak | R1 is a bounded in-memory test server; no hour-long multi-service soak or production load claim. |
| Real money, credentials, PII, or external integrations | Explicitly out of scope; all data and examples are synthetic. |

Playback R1 is evidence against the production router on loopback, not a live standalone binary or platform-soak result.

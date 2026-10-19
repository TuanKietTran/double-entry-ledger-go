# Double-entry ledger

CQRS system of record for synthetic financial journals. See [`api/openapi.yaml`](../api/openapi.yaml) for the callable API contract and [`openspec/dod.md`](dod.md) for measured status and limitations.

## Stack

Go 1.25+, Chi, `shopspring/decimal`, Google UUID, pgx v5, and embedded PostgreSQL tests.

## Public API

Accounts, transactions, reversals, entry pages, balances/as-of, trial balance, and transaction lookup. Default port 8101. Explicit `offset` remains supported; cursor pagination is available through `X-Next-Cursor`.

## Checkpoints

| Date | Change | Commit | Evidence |
|---|---|---|---|
| 2026-10-17 | Handler tests and PostgreSQL repository | `a28e5fb` | Embedded PostgreSQL tests. |
| 2026-10-17 | Legacy 1,000-request playback | existing baseline | `TestThousandPerMinuteAndPlayback`; regression evidence, not real-process throughput. |
| 2026-10-18 | Ledger contract and implementation | `31f6879` | Race suite and OpenAPI route/status checks. |
| 2026-10-19 | LF policy | `1e43baa` | `gofmt -l .`, build, vet, race suite, and tidy gates. |
| 2026-10-19 | Production-router 10k fault replay | `bc1a125` | `TestR1TenThousandProductionHTTPRequests`; full race suite. |
| 2026-10-19 | Bounded default playback concurrency | `c741054` | Five repeated legacy replay passes and full race suite. |
| 2026-10-19 | Cursor pagination | `d64a579` | API concurrent-append and PostgreSQL timestamp-tie cursor tests; full race suite. |
| 2026-10-19 | R1 and cursor specifications | `d64a579` | `go test -p 2 ./... -race -count=1 -cover` passed; the named R1 and cursor tests cover the behavior. |

## Scope notes

No downstream service is consumed. Durable production operations, remote-database fault recovery, hosted CI execution, exhaustive OpenAPI semantics, and sustained live-process load remain unverified or out of scope. See the DoD.

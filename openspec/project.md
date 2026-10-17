# Double-entry ledger

CQRS system of record for synthetic financial journals. See [`api/openapi.yaml`](../api/openapi.yaml) for callable API contract and [`openspec/dod.md`](dod.md) for measured status and limitations.

## Stack

Go 1.22, Chi, `shopspring/decimal`, Google UUID, `jackc/pgx` v5, embedded Postgres tests.

## Public API

See OpenAPI contract: accounts, transactions, reversal, entries, balances/as-of, trial balance, transaction lookup. Default port 8101.

## Checkpoints

| Date | Change | Evidence |
|---|---|---|
| 2026-10-17 | Handler tests and Postgres repository | `a28e5fb`; embedded Postgres tests. |
| 2026-10-17 | Playback baseline | 1000 posts plus retry, 83ms in playback test; regression evidence only, not real-process throughput. |
| 2026-10-18 | Production ledger contract and implementation | `31f6879`; `go test -p 2 ./... -race -count=1 -cover` passed; package coverage cmd 43.1%, API 80.8%, command 94.4%, domain 93.7%, playback 63.5%, query 100.0%, repository 83.5%. Weighted internal 84.0% measured at card 5d. OpenAPI parsed and route/status contract test passes. |

## Scope notes

No downstream service is consumed. Cursor pagination, live process probes, and exhaustive OpenAPI schema validation remain incomplete; see DoD. See `openspec/gifs/playback-flow.gif` as legacy storyboard only.

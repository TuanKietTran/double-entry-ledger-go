# Double-entry ledger

CQRS system of record for synthetic financial journals. See [`api/openapi.yaml`](../api/openapi.yaml) for the callable API contract and [`openspec/dod.md`](dod.md) for measured status and limitations.

## Stack

Go 1.25+, Chi, `shopspring/decimal`, Google UUID, pgx v5, and embedded PostgreSQL tests.

## Public API

Accounts, transactions, reversals, entry pages, balances/as-of, trial balance, and transaction lookup. Default port 8101. Explicit `offset` remains supported; cursor pagination is available through `X-Next-Cursor`.

## Scope notes

No downstream service is consumed. Cursor continuation is keyset-based but not a snapshot: an append that sorts before the cursor can be omitted from the continuation. Durable production operations, remote-database fault recovery, hosted CI execution, exhaustive OpenAPI semantics, and sustained live-process load remain unverified or out of scope. See the DoD.

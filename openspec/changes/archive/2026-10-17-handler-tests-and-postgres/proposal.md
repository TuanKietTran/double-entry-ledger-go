# Handler tests and Postgres repository

## Why

The ledger had an in-memory CQRS path and a SQL migration, and neither handler tests nor a real Postgres store.

## What

- Unit tests for balanced posts, unbalanced rejection, idempotency, and a zero balance.
- `PostgresLedger` against `migrations/001_init.sql`.
- `DATABASE_URL` selects Postgres; unset stays in memory.

## Impact

- Specs: `post-transaction`, `account-balance` (recorded as the current contract), `persistence` (added).
- No HTTP route added or removed.

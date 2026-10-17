# Ledger persistence design

- `repository.LedgerRepository` is the port: `SaveTransaction`, `GetBalance`, `GetEntries`.
- `NewInMemoryLedger` is the default. `NewPostgresLedger` is used when `DATABASE_URL` is set. The caller applies `migrations/001_init.sql`; the server does not migrate on startup.
- Balance is the sum of `DEBIT` amounts minus `CREDIT` amounts. An account with no entries returns zero and `USD`.
- Postgres inserts a missing account as `owner_id = unspecified`, `type = ASSET`, `status = ACTIVE`, currency copied from the entry. A conflict on `id` leaves the existing row.
- Idempotency is `transactions.idempotency_key`. A conflict returns `idempotency key already used: <existing uuid>` and does not insert entries.
- Amounts are `NUMERIC` via `shopspring/decimal`. The handler rejects an unbalanced transaction before the repository is called.
- Handler tests cover the in-memory port. `TestPostgresLedger` runs the same rules on embedded Postgres 16 and skips when the test process is root, because `initdb` refuses root.

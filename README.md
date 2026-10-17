# Double-Entry Ledger (CQRS)

A double-entry ledger service built in Go with CQRS.

## Structure
- `internal/domain` — aggregates and value objects
- `internal/command` — command handlers (writes)
- `internal/query` — query handlers (reads)
- `internal/repository` — persistence interfaces
- `cmd/server` — entrypoint

## Commands
- CreateAccount
- PostTransaction

## Queries
- GetAccountBalance
- GetAccountEntries

## Run
```bash
go test ./...
go run ./cmd/server
```

`DATABASE_URL` selects the store. Unset uses the in-memory repository. When set, the server uses Postgres and expects `migrations/001_init.sql` to already be applied. Referenced accounts are created on first post if they do not exist.

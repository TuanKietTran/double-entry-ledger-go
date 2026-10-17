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
go run ./cmd/server
```

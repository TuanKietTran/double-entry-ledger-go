# Tasks

## 1. Handler tests

- [x] Balanced posts update both balances
- [x] An unbalanced post is rejected and does not change balances
- [x] A reused idempotency key errors with the original id and does not double post
- [x] An unknown account reads as zero USD
- [x] Repository errors propagate from the handlers

## 2. Postgres repository

- [x] Save a transaction, including a missing account row
- [x] Enforce idempotency inside the database transaction
- [x] Read balance and entries in post order
- [x] Select the store from `DATABASE_URL` in `cmd/server`

## 3. Verification

- [x] `go test ./...`
- [x] Embedded Postgres 16: balance 59.75, idempotency, unknown account

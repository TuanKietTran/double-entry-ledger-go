# Double-entry ledger

CQRS ledger. Callers post balanced entries and read an account balance (debits minus credits).

## Stack

Go 1.22, Chi, `shopspring/decimal`, Google UUID. Postgres via `jackc/pgx` v5 when configured.

## Public API

- `POST /v1/transactions`
- `GET /v1/accounts/{id}/balance`

## Checkpoints

| Date | Change | Evidence |
| --- | --- | --- |
| 2026-10-17 | Handler tests and Postgres repository | `a28e5fb`. `go test ./...`. Embedded Postgres 16 covered balance, idempotency, and an unknown account. |
| 2026-10-17 | 1000 requests in one minute, retry, playback, coverage | `internal/playback`. 1000 posts plus one 503 retry each finished in 83ms. Replay did not change the balance. Statement coverage 50.2% (`go test -coverpkg=./...`). |
| 2026-10-17 | Definition of done and playback storyboard | `openspec/dod.md` and `openspec/gifs/playback-flow.gif`. GIF review: retry stays at the pre-write state, playback shows the unchanged result. |

## Out of scope until implemented

- `CreateAccount` as its own command. A missing account row is created on post when Postgres is used.
- `GetAccountEntries` over HTTP. Entries are available on the repository port only.

## Definition of done

See `openspec/dod.md`. The playback storyboard is `openspec/gifs/playback-flow.gif`.

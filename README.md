# Double-Entry Ledger

A synthetic-only ledger service demonstrating exact-money journals, explicit account provisioning, append-only corrections, idempotent writes, and CQRS reads. It processes no real funds, PSP traffic, credentials, or PII.

## Architecture

```mermaid
flowchart LR
  Caller --> API[Chi HTTP API :8101]
  API --> C[Command handlers]
  API --> Q[Query handlers]
  C --> D[Domain invariants]
  C --> R[Repository port]
  Q --> R
  R --> M[Memory adapter]
  R --> P[Postgres adapter]
```

`cmd/server` wires `internal/api`, `command`, `query`, domain, and repository adapters. Reads and writes share the repository boundary; there is no downstream service dependency.

## Run

Go 1.22+; Docker is not required. With `DATABASE_URL` unset the server uses memory. With it set the server uses Postgres and applies the SQL migrations before serving. Default listener is `:8101`; override with `HTTP_ADDR`.

```sh
go test ./...
go run ./cmd/server
```

Create two accounts, then post a balanced exact-decimal transaction (use UUIDs you provision):

```sh
curl -i localhost:8101/v1/accounts -H 'Content-Type: application/json' -d '{"account_id":"11111111-1111-4111-8111-111111111111","currency":"USD","type":"ASSET"}'
curl -i localhost:8101/v1/accounts -H 'Content-Type: application/json' -d '{"account_id":"22222222-2222-4222-8222-222222222222","currency":"USD","type":"EQUITY"}'
curl -i localhost:8101/v1/transactions -H 'Content-Type: application/json' -d '{"idempotency_key":"demo-001","entries":[{"account_id":"11111111-1111-4111-8111-111111111111","amount":"12.34","direction":"DEBIT","currency":"USD"},{"account_id":"22222222-2222-4222-8222-222222222222","amount":"12.34","direction":"CREDIT","currency":"USD"}]}'
curl -i localhost:8101/v1/accounts/11111111-1111-4111-8111-111111111111/balance
```

Full request/response shapes and statuses are authoritative in [`api/openapi.yaml`](api/openapi.yaml). Writes require explicit account provisioning. Same idempotency key and same payload replays the original result; a changed payload conflicts. Optional `expected_versions` checks account versions.

## API and operations

Current routes: account provision/read, transaction post/lookup by ID or idempotency key, reversal, account entries and current/as-of balance, trial balance. `/healthz`, `/readyz`, and `/metrics` are operational endpoints. HTTP errors use a JSON envelope and `X-Request-ID`. `DATABASE_URL` is optional; `HTTP_ADDR` defaults to `:8101`.

## Glossary

- **Journal/transaction:** immutable group of entries whose debits equal credits.
- **Entry:** positive exact decimal amount, account, currency, and strict DEBIT/CREDIT direction.
- **Reversal:** a new transaction with inverted legs, preserving the original.
- **Idempotency key:** caller identity for a write, bound to its canonical payload.
- **Trial balance:** per-account debit-minus-credit total; unlike currencies are not combined.

## Measured status

Evidence from `go test -p 2 ./... -race -count=1 -cover` on 2026-10-18 in this Go environment: API 80.8%, command 94.4%, domain 93.7%, playback 63.5%, query 100.0%, repository 83.5%, cmd 43.1%. The repository package reports pass, but the embedded Postgres test explicitly skips when run as root; PostgreSQL parity was not verified in this environment. These package percentages are not an aggregate; weighted internal coverage was measured at 84.0% with `-coverpkg=./internal/...`.

## Design decisions

See [ADRs](docs/adr/). This repository demonstrates domain-first money invariants, CQRS boundaries, payload-bound idempotency, optimistic versions, and memory/Postgres adapter parity; it does not claim production deployment, external-service resilience, or independently measured live-process performance.

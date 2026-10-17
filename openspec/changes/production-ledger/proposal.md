# Production ledger contract and invariants

## Why
The service is intended to be the system of record for simulated financial flows, but currently it only exposes transaction posting and a current balance. Existing validation accepts empty journals and unknown directions inconsistently, and account provisioning is implicit in Postgres.

## Scope
Publish an accurate contract for current routes while specifying explicit account lifecycle, strict one-currency balanced positive journals, append-only reversal, query, payload-bound idempotency, concurrency and memory/Postgres parity as planned capabilities. Existing route behavior will be evolved only with tests and a migration note. This proposal does not claim any planned route is implemented.

## Non-goals
Real funds, external service calls, real personal data, or unverified durability claims.

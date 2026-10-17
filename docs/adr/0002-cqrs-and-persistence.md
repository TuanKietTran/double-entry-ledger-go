# ADR 0002: CQRS with repository adapters

- Status: Accepted
- Decision: Keep command and query handlers separate over repository ports with in-memory and PostgreSQL adapters.
- Rationale: Isolates domain behavior from transport and persistence while allowing deterministic tests and SQL parity.
- Consequence: Both adapters must preserve the same atomicity, idempotency and concurrency contract.

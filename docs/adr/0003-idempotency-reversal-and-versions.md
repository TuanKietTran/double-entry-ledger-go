# ADR 0003: Idempotent writes, append-only reversal, optimistic versions

- Status: Accepted
- Decision: Bind write idempotency keys to canonical payloads, represent corrections as new reversal transactions, and optionally compare account versions on post.
- Rationale: Retries must not duplicate effects; immutable history is auditable; expected versions detect stale writers.
- Consequence: Changed payload under an existing key and stale versions conflict. A transaction supports one full reversal; partial corrections are separate balanced journals.

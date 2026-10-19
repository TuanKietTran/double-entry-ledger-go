# Platform 03: Idempotency

- Status: Accepted
- Decision: Bind each write key to its canonical payload and return the original transaction on an exact retry.
- Rationale: A caller can safely retry after an ambiguous response without applying a journal twice.
- Consequence: Reusing a key with a different payload conflicts; reversals are new append-only journals with their own keys.

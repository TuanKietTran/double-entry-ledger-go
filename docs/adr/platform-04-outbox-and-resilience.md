# Platform 04: Outbox and resilience

- Status: Accepted
- Decision: Use an outbox when a local write must trigger an external side effect. Not applicable to this ledger: it makes no downstream calls and has no outbox.
- Rationale: A local database transaction cannot atomically commit a remote service operation.
- Consequence: The playback client retries only transient HTTP/transport faults with bounded full-jitter exponential backoff; idempotency protects repeated posts. This is a test client policy, not a downstream resilience claim.

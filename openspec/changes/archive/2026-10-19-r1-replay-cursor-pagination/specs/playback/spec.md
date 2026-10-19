## ADDED Requirements

### Requirement: Complete 10,000 production-router HTTP calls within one minute
The R1 test SHALL submit 10,000 logical calls through the production HTTP router over loopback, with bounded concurrency, at least 5% injected 503 responses, some 429 responses, 1% attempt timeouts, and at least 2% identical same-key/body duplicates. It SHALL finish all calls in a 60-second window, retrying transient failures with capped exponential full jitter, a 50 ms base, a 2 second maximum delay, and no more than six attempts.

#### Scenario: Retry transient outcomes without duplicate effects
- **GIVEN** the real router serves the in-memory ledger and fault injection returns retryable errors for selected calls
- **WHEN** 10,000 submissions are replayed, including repeated keys with identical payloads
- **THEN** every call reaches HTTP success within the window
- **AND** exact duplicate submissions return the same transaction identity without advancing account versions again
- **AND** the balance fold and trial-balance invariant hold

#### Scenario: Per-attempt timeout with live overall window
- **GIVEN** one attempt exceeds its own deadline while the overall replay context remains live
- **WHEN** that request is retried
- **THEN** the per-attempt timeout is retryable
- **AND** cancellation of the overall replay context remains final

# Playback

## Purpose

Playback exercises balanced ledger writes through HTTP with retryable faults and payload-bound idempotency. A legacy 1,000-request fixture remains as regression evidence; the R1 target is 10,000 requests against the production router.

## Requirements

### Requirement: Preserve legacy playback regression
The client SHALL keep the 1,000-call fixture and verify that repeating the same call list does not change stored totals.

#### Scenario: Legacy first and second play
- **GIVEN** balanced one-dollar posts with distinct idempotency keys
- **WHEN** the client plays 1,000 calls and plays the same list again
- **THEN** each first-play request succeeds after the injected transient failure
- **AND** replay returns the original transaction result without changing balances

### Requirement: Complete 10,000 production-router HTTP calls within one minute
The R1 test SHALL submit 10,000 logical calls through the production HTTP router over loopback, with bounded concurrency, at least 5% injected 503 responses, some 429 responses, 1% attempt timeouts, and at least 2% identical same-key/body duplicates. It SHALL finish all calls in a 60-second window, retrying transient failures with capped exponential full jitter, a 50 ms base, a 2 second maximum delay, and no more than six attempts.

#### Scenario: Retry transient outcomes without duplicate effects
- **GIVEN** the real router serves the in-memory ledger and fault injection returns retryable errors for selected calls
- **WHEN** 10,000 submissions are replayed, including repeated keys with identical payloads
- **THEN** every call reaches HTTP success within the window
- **AND** no business 4xx response is retried
- **AND** exact duplicate submissions return the same transaction identity without advancing account versions again
- **AND** the balance fold and trial-balance invariant hold

#### Scenario: Per-attempt timeout with live overall window
- **GIVEN** one attempt exceeds its own deadline while the overall replay context remains live
- **WHEN** that request is retried
- **THEN** the per-attempt timeout is retryable
- **AND** cancellation of the overall replay context remains final

### Requirement: Retry only transient failures
The client SHALL retry transport errors and HTTP 408, 429, or 5xx, and SHALL NOT retry other response statuses.

#### Scenario: HTTP 400 is final
- **WHEN** the server returns HTTP 400
- **THEN** the client stops after one attempt

#### Scenario: Repeated 503 stops at the attempt limit
- **WHEN** the server returns HTTP 503 on every attempt
- **THEN** the client stops after at most six attempts
- **AND** the last outcome is terminal and unsuccessful

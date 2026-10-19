# Playback design

- Package `internal/playback`. `Play` uses a bounded worker pool and preserves input ordering in its outcome slice.
- The default policy is six maximum attempts, 50 ms exponential-backoff base, 2 second delay cap, bounded concurrency, and per-attempt contexts. Delay is independently selected uniformly from zero to the current exponential cap (full jitter).
- Retryable statuses are 408, 429, and 5xx. Transport and per-attempt timeout failures retry while the overall context remains live; overall cancellation stops playback.
- `TestThousandPerMinuteAndPlayback` preserves the legacy 1,000-call regression. `TestR1TenThousandProductionHTTPRequests` submits 10,000 requests to `internal/serverruntime.BuildRouter` over loopback and verifies faults, duplicate effect suppression, account versions, and trial balance.

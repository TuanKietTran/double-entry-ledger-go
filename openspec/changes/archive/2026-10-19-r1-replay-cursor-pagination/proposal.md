# R1 HTTP replay and stable entry cursors

## Why

The prior replay only exercised 1,000 sequential calls and did not establish retry behavior under realistic transient HTTP faults. Offset pagination could shift when new entries arrived between pages.

## Scope

Use bounded concurrent playback against the production router with an explicit retry budget and fault injection. Add a stable `(created_at, entry_id)` cursor to entry reads while retaining explicit offset parameters. Keep the standalone HTTP server loopback-only.

## Outcome

Implemented and covered by the R1 replay test, cursor API test, and PostgreSQL timestamp-tie test. This does not claim a live binary load test or sustained production throughput.

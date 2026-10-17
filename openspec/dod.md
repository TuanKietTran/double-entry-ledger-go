# Definition of done

A box is **Met** only when the code behaves that way and a test asserts it. **Partial** means the code does it, but no test asserts that scenario. **Unmet** means it is not implemented.

Playback storyboard: `openspec/gifs/playback-flow.gif`.

## Met

| Done when | Evidence |
| --- | --- |
| A balanced post returns a transaction id and moves both balances | `internal/command` handler test. Playback balance goes from $0 to $1, then to $1000. |
| An unbalanced post is HTTP 400 and changes nothing | Handler test. |
| A reused idempotency key is HTTP 400 with the original id and does not post again | Handler test and playback frame 6. Balance stays $1000. |
| An account with no entries reads as 0 USD | Handler test and Postgres test. |
| `DATABASE_URL` unset uses memory. Set, it uses Postgres and the same balance and idempotency rules | `cmd/server` and `TestPostgresLedger`. |
| A missing Postgres account row is created on post | Postgres test. |
| 1000 calls finish in under 60 seconds, with one HTTP 503 retried on each call | Playback test, 83ms. |
| HTTP 408, 429, and 5xx retry up to 3 times. HTTP 400 does not retry | `TestRetriesTransientFailure`, `TestDoesNotRetryClientError`, `TestStopsAfterMaxAttempts`. |
| Statement coverage is measured and written on the checkpoint | 50.2% from `go test -coverpkg=./...`. |

## Partial

| Done when | Why it is not Met |
| --- | --- |
| Invalid JSON and a non-UUID account id are HTTP 400 | Implemented in the handler. No test sends those bodies. |
| Only the token `CREDIT` counts as a credit | Implemented. The tests use `DEBIT` and `CREDIT` and never a lowercase token. |
| An empty entry list is accepted | `IsBalanced` treats 0 as 0. No test posts an empty list. |
| Currency is the latest entry's currency | The USD path is tested. An EUR entry is not. |

## Unmet

| Not done | Why |
| --- | --- |
| `CreateAccount` as its own command | A missing row is created only as a side effect of a Postgres post. |
| `GetAccountEntries` over HTTP | Entries exist on the repository port only. |
| Coverage at a high bar | 50.2% is the measured total. `cmd/server` and some branches are untested. There is no 80% gate. |

## GIF review

Good enough to show the flow. Frame 3 keeps the balance at $0 while the retry is in flight. Frame 6 returns HTTP 400 and the balance stays $1000. It is a storyboard of the test, not a screen capture.

## ADDED Requirements

### Requirement: Current HTTP contract is explicit
The service SHALL publish an OpenAPI contract that lists only implemented routes as callable operations and their actual response statuses. Planned routes SHALL be identified as not implemented and SHALL NOT be represented as callable paths.

#### Scenario: Contract matches current routes
- **GIVEN** the server's current route registration
- **WHEN** the contract validation test runs
- **THEN** every registered route and method appears in OpenAPI with its implemented success and error statuses
- **AND** no planned-only route appears under `paths`

### Requirement: Published response shapes
Every implemented operation MUST declare an application/json success schema and every declared error status MUST reference the shared error envelope.

#### Scenario: Handler output matches the contract
- **GIVEN** provisioned accounts and a balanced transaction
- **WHEN** posting, looking up, reversing, or querying through the HTTP handlers
- **THEN** representative responses contain the documented required properties with the declared JSON types
- **AND** representative invalid requests return the shared error envelope.

### Requirement: Explicit account and journal model
The ledger SHALL provision accounts explicitly with immutable type and currency and an explicit active status. A posted transaction SHALL contain at least two positive exact-amount entries, use recognized directions, and use one explicit currency across all legs. The sum of debit amounts SHALL equal the sum of credit amounts.

#### Scenario: Reject invalid journals without mutation
- **GIVEN** an empty, non-positive, unknown-direction, multi-currency, or unbalanced journal
- **WHEN** it is posted
- **THEN** the ledger rejects it and no account balance or journal changes

### Requirement: History is append-only
The ledger SHALL expose reversals as new transactions referencing the original and SHALL NOT edit or delete posted entries. Corrections SHALL be represented by new journal activity.

### Requirement: Queryable ledger history
The ledger SHALL support paginated account entries, as-of account balances, trial balance, and transaction lookup by identity and idempotency key. Balance results SHALL equal the debit-minus-credit fold of applicable entries.

### Requirement: Loopback-only HTTP listener
The server SHALL bind only to loopback addresses and SHALL reject non-loopback `HTTP_ADDR` values. The default address SHALL be `127.0.0.1:8101`.

#### Scenario: Reject a public bind address
- **WHEN** `HTTP_ADDR` is configured as a wildcard or non-loopback IP
- **THEN** configuration validation fails before listening

### Requirement: Stable cursor entry pagination
The ledger SHALL support exclusive keyset pagination ordered by `(created_at, entry_id)` and SHALL return an opaque continuation cursor when another page exists. Explicit offset pagination SHALL remain available for compatibility.

#### Scenario: Continue after a later append
- **GIVEN** a caller has received a cursor for an entry page
- **AND** additional entries are appended after that page was read
- **WHEN** the caller requests the next page using the cursor
- **THEN** entries are ordered after the cursor key without repeating earlier entries
- **AND** newly appended entries after the cursor key are visible

#### Scenario: Stable timestamp ties
- **GIVEN** multiple entries share a creation timestamp
- **WHEN** cursor pages are read
- **THEN** entry UUID provides a deterministic tie-break and concatenated pages match that order

### Requirement: Idempotent concurrent writes
A write key SHALL bind to a canonical payload. Repeating the same key and payload SHALL return the original result without a second effect; reusing it with a different payload SHALL conflict. Concurrent journal writes SHALL preserve balance and account-version invariants.

### Requirement: Memory and Postgres parity
Memory and Postgres adapters SHALL implement the same account, transaction, idempotency, conflict, and query semantics, with atomic writes.

### Requirement: Independent connection conflict safety
The PostgreSQL adapter MUST preserve journal folds and account versions when independent connections submit concurrent replay, conflicting payloads, stale versions or reversals.

#### Scenario: Conflicting concurrent writes
- **GIVEN** explicitly provisioned accounts and independent PostgreSQL connections
- **WHEN** concurrent callers submit a changed-payload key conflict or the same expected account version
- **THEN** exactly one write commits and the losing write changes neither entries nor account versions.

#### Scenario: Tied entry timestamps
- **GIVEN** account entries with the same creation timestamp
- **WHEN** offset pages are read without concurrent appends
- **THEN** the page concatenation equals the full result ordered by timestamp then entry UUID.

### Requirement: Explicit account provisioning
Clients SHALL provision accounts through `POST /v1/accounts` before posting to them. Repeating the identical account identity and immutable attributes SHALL succeed without creating a duplicate; reusing an identity with different owner, currency, type, or status SHALL conflict. Posting SHALL NOT create missing account rows as a side effect.

#### Scenario: Replace legacy implicit provisioning
- **GIVEN** a transaction references an account that has not been explicitly provisioned
- **WHEN** the transaction is posted
- **THEN** the post is rejected without creating an account or transaction
- **AND** clients can provision the account explicitly and retry the post.

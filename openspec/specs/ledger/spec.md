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

### Requirement: Idempotent concurrent writes
A write key SHALL bind to a canonical payload. Repeating the same key and payload SHALL return the original result without a second effect; reusing it with a different payload SHALL conflict. Concurrent journal writes SHALL preserve balance and account-version invariants.

### Requirement: Memory and Postgres parity
Memory and Postgres adapters SHALL implement the same account, transaction, idempotency, conflict, and query semantics, with atomic writes.

### Requirement: Explicit account provisioning
Clients SHALL provision accounts through `POST /v1/accounts` before posting to them. Repeating the identical account identity and immutable attributes SHALL succeed without creating a duplicate; reusing an identity with different owner, currency, type, or status SHALL conflict. Posting SHALL NOT create missing account rows as a side effect.

#### Scenario: Replace legacy implicit provisioning
- **GIVEN** a transaction references an account that has not been explicitly provisioned
- **WHEN** the transaction is posted
- **THEN** the post is rejected without creating an account or transaction
- **AND** clients can provision the account explicitly and retry the post.

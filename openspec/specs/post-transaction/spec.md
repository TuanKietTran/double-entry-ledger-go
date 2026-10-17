# Post transaction

## Purpose

Record a double-entry movement once, and refuse it when the sides do not match or the idempotency key was already used.

## Requirements

### Requirement: Accept a balanced post
The system SHALL accept a transaction whose debit amounts equal its credit amounts and return a new transaction id.

#### Scenario: Two-sided post
- **GIVEN** idempotency key `txn-fund-1` has not been used
- **WHEN** a client posts a 100 USD debit to account A and a 100 USD credit to account B with that key
- **THEN** the response is HTTP 200 and contains `transaction_id`
- **AND** the movement is stored once

#### Scenario: Empty entry list
- **WHEN** a client posts a transaction with no entries
- **THEN** the system treats the two sides as equal and accepts the post

### Requirement: Reject an unbalanced post
The system SHALL reject a transaction whose debit total does not equal its credit total, and SHALL NOT change any balance.

#### Scenario: Debit exceeds credit
- **WHEN** a client posts a 10 USD debit and a 9 USD credit
- **THEN** the response is HTTP 400
- **AND** the body contains `not balanced`
- **AND** both accounts still show a zero balance

### Requirement: Idempotency key binds to its payload
The system SHALL require a nonempty idempotency key and bind it to the transaction description and journal entries. A retry with the same payload SHALL return the original transaction id without another effect. Reuse with a different payload SHALL return HTTP 409 and SHALL NOT change the ledger.

#### Scenario: Same key and payload replay
- **GIVEN** a balanced post with key `same-key` succeeded and returned transaction id T
- **WHEN** the client posts the same payload again with key `same-key`
- **THEN** the response is HTTP 200 with transaction id T
- **AND** each account still has only the entries from the first post

#### Scenario: Same key with changed payload
- **GIVEN** a balanced post with key `same-key` succeeded
- **WHEN** the client reuses the key with a different description or journal payload
- **THEN** the response is HTTP 409
- **AND** no second transaction is recorded

### Requirement: Reject a malformed post
The system SHALL reject a body it cannot apply, before any balance changes.

#### Scenario: Invalid JSON
- **WHEN** the body is not JSON
- **THEN** the response is HTTP 400 and the body contains `invalid json`

#### Scenario: Account id is not a UUID
- **WHEN** an entry `account_id` is not a UUID
- **THEN** the response is HTTP 400 and the body contains `invalid account id`

### Requirement: Reversal is an append-only full journal
The system SHALL reverse a posted transaction by appending a new transaction with the same accounts, amounts, and currencies and the opposite direction for every entry. Each original transaction SHALL have at most one reversal; reversing a reversal SHALL be rejected.

#### Scenario: Full reversal preserves history
- **GIVEN** a posted transaction T
- **WHEN** a caller reverses T with an idempotency key
- **THEN** a new transaction references T and contains opposite legs
- **AND** T and its original entries remain unchanged
- **AND** replaying the same reversal key returns the same reversal transaction id

### Requirement: Direction token
The system SHALL accept only the exact direction tokens `DEBIT` and `CREDIT`.

#### Scenario: Exact credit token
- **WHEN** an entry direction is `CREDIT`
- **THEN** the amount counts toward the credit total

#### Scenario: Unknown direction
- **WHEN** an entry direction is not exactly `DEBIT` or `CREDIT`
- **THEN** the request is rejected with HTTP 400 before persistence

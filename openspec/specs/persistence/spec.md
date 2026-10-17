# Ledger persistence

## Purpose

Keep the post and balance rules on the store selected at startup.

## Requirements

### Requirement: In-memory store by default
The system SHALL keep posts in process memory when `DATABASE_URL` is unset.

#### Scenario: Balance follows a post in the same process
- **GIVEN** `DATABASE_URL` is unset
- **WHEN** a client posts a balanced transaction and then reads a balance
- **THEN** the balance includes that post
- **AND** the data is gone when the process exits

### Requirement: Postgres store when configured
The system SHALL use Postgres when `DATABASE_URL` is set, and SHALL apply the same balance and idempotency rules there.

#### Scenario: Post survives a new connection
- **GIVEN** `DATABASE_URL` points at a database where `migrations/001_init.sql` has been applied
- **WHEN** a client posts a balanced transaction
- **THEN** a later balance read through a new connection returns that post's effect

#### Scenario: Duplicate key does not double post
- **GIVEN** the Postgres store already holds a post for an idempotency key
- **WHEN** a second post uses that key
- **THEN** the store rejects it with `idempotency key already used` and the original transaction id
- **AND** the account still has only the first post's entries

### Requirement: Accounts are provisioned explicitly
The system SHALL require account records to exist before a post references them. A rejected post SHALL NOT create accounts or journal rows as a side effect.

#### Scenario: Missing account is rejected
- **GIVEN** the store has no row for account A
- **WHEN** a balanced post includes A
- **THEN** the post is rejected
- **AND** no account or entry is created

#### Scenario: Provision before first post
- **GIVEN** account A has been explicitly provisioned with a currency and type
- **WHEN** a balanced post references A using its provisioned currency
- **THEN** the post succeeds

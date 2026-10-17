# Account balance

## Purpose

Report one account's net balance as debits minus credits.

## Requirements

### Requirement: Net of posted entries
The system SHALL return the requested account id, the sum of debit amounts minus the sum of credit amounts, and a currency.

#### Scenario: Debit then partial credit
- **GIVEN** account A was debited 100 USD and later credited 40.25 USD by balanced posts
- **WHEN** a client gets `/v1/accounts/{A}/balance`
- **THEN** the response is HTTP 200
- **AND** `Available` equals 59.75
- **AND** `Currency` is `USD`
- **AND** `AccountID` is A

#### Scenario: Credit-normal counterparty
- **GIVEN** the offsetting credits and debits of the scenario above landed on account B
- **WHEN** a client gets B's balance
- **THEN** `Available` equals -59.75

### Requirement: Unknown account
The system SHALL answer an account that has no entries with a zero balance in USD.

#### Scenario: No entries
- **WHEN** a client gets the balance of an account that has never been posted
- **THEN** the response is HTTP 200
- **AND** `Available` is 0
- **AND** `Currency` is `USD`

### Requirement: Invalid account id
The system SHALL reject a balance read whose path id is not a UUID.

#### Scenario: Not a UUID
- **WHEN** the path id is not a UUID
- **THEN** the response is HTTP 400 and the body contains `invalid account id`

### Requirement: Currency of the latest entry
The system SHALL report the currency of the account's latest entry.

#### Scenario: Entries in one currency
- **GIVEN** every entry on the account is EUR
- **WHEN** the client reads the balance
- **THEN** `Currency` is `EUR`

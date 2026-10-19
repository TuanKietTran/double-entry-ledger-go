## ADDED Requirements

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

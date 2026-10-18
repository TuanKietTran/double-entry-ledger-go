## ADDED Requirements

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

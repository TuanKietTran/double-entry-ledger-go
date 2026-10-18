## ADDED Requirements

### Requirement: Published response shapes
Every implemented operation MUST declare an application/json success schema and every declared error status MUST reference the shared error envelope.

#### Scenario: Handler output matches the contract
- **GIVEN** provisioned accounts and a balanced transaction
- **WHEN** posting, looking up, reversing, or querying through the HTTP handlers
- **THEN** representative responses contain the documented required properties with the declared JSON types
- **AND** representative invalid requests return the shared error envelope.

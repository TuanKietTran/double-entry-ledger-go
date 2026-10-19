# Platform 05: Persistence and testing

- Status: Accepted
- Decision: Keep memory and PostgreSQL implementations behind the ledger repository interface; run database integration tests with embedded PostgreSQL.
- Rationale: Memory enables fast deterministic tests, while PostgreSQL exercises SQL transactions, constraints, versions, and concurrent connections without requiring Docker.
- Consequence: Adapter parity is tested, but embedded-database results do not establish production operations or remote database fault recovery.

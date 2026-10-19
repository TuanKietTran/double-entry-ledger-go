# Platform 01: CQRS

- Status: Accepted
- Decision: Separate command and query handlers behind repository interfaces.
- Rationale: Keep journal invariants in the domain and permit focused read models without transport or database coupling.
- Consequence: Adapters and handlers are tested independently; they share the same persisted journal state.

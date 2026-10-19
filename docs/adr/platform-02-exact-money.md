# Platform 02: Exact money

- Status: Accepted
- Decision: Use exact decimal arithmetic, positive amounts, explicit currency, and balanced single-currency journals.
- Rationale: Binary floating point and implicit currency conversion are unsuitable for ledger invariants.
- Consequence: Cross-currency posting and FX conversion are outside this API.

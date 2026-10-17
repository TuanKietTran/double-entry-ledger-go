# ADR 0001: Exact money and journal invariants

- Status: Accepted
- Decision: Keep amounts as exact decimal strings/decimal values, require positive amounts, explicit currency and strict direction, and validate each journal's debit total equals its credit total.
- Rationale: Binary floating point is unsuitable for money; a one-currency journal avoids implicit FX semantics.
- Consequence: Cross-currency journals and conversion are out of scope; callers must provision accounts.

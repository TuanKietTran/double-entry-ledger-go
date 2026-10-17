-- Double-entry ledger schema
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    owner_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    idempotency_key TEXT UNIQUE NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    reversal_of UUID REFERENCES transactions(id),
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    amount NUMERIC(20,6) NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('DEBIT', 'CREDIT')),
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_transactions_single_reversal ON transactions(reversal_of) WHERE reversal_of IS NOT NULL;

CREATE INDEX idx_entries_account ON entries(account_id);
CREATE INDEX idx_entries_tx ON entries(transaction_id);

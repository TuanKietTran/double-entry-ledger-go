-- Additive migration: safe for databases that applied 001 before reversal support.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS reversal_of UUID REFERENCES transactions(id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_single_reversal
    ON transactions(reversal_of) WHERE reversal_of IS NOT NULL;

package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Compile-time checks that both stores satisfy the command/query port.
var (
	_ LedgerRepository  = (*PostgresLedger)(nil)
	_ LedgerRepository  = (*InMemoryLedger)(nil)
	_ AccountRepository = (*PostgresLedger)(nil)
	_ AccountRepository = (*InMemoryLedger)(nil)
)

// PostgresLedger persists transactions against migrations/001_init.sql.
// Accounts must be explicitly provisioned before a transaction can reference them.
type PostgresLedger struct {
	pool *pgxpool.Pool
}

func NewPostgresLedger(ctx context.Context, dsn string) (*PostgresLedger, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &PostgresLedger{pool: pool}, nil
}

func (r *PostgresLedger) Close() {
	r.pool.Close()
}

func (r *PostgresLedger) CreateAccount(ctx context.Context, account domain.Account) error {
	if err := account.Validate(); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounts (id, owner_id, currency, type, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, account.ID, account.OwnerID, account.Currency, string(account.Type), string(account.Status))
	if err != nil {
		return err
	}
	existing, err := r.GetAccount(ctx, account.ID)
	if err != nil {
		return err
	}
	if !existing.SameDefinition(account) {
		return fmt.Errorf("account already exists with different attributes")
	}
	return nil
}

func (r *PostgresLedger) GetAccount(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	var account domain.Account
	var accountType, status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, currency, type, status, version FROM accounts WHERE id = $1
	`, id).Scan(&account.ID, &account.OwnerID, &account.Currency, &accountType, &status, &account.Version)
	if err != nil {
		return domain.Account{}, err
	}
	account.Type = domain.AccountType(accountType)
	account.Status = domain.AccountStatus(status)
	return account, nil
}

func (r *PostgresLedger) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	if tx.ReversalOf != uuid.Nil {
		return fmt.Errorf("reversal transactions must use SaveReversal")
	}
	return r.save(ctx, tx, uuid.Nil)
}

func (r *PostgresLedger) SaveReversal(ctx context.Context, originalID domain.TransactionID, tx domain.Transaction) error {
	if originalID == uuid.Nil || tx.ReversalOf != originalID {
		return fmt.Errorf("reversal reference does not match original transaction")
	}
	return r.save(ctx, tx, originalID)
}

func (r *PostgresLedger) save(ctx context.Context, tx domain.Transaction, reversalOf domain.TransactionID) error {
	if err := tx.Validate(); err != nil {
		return err
	}
	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	if reversalOf != uuid.Nil {
		var originalReversal uuid.UUID
		if err := dbTx.QueryRow(ctx, `SELECT COALESCE(reversal_of, '00000000-0000-0000-0000-000000000000'::uuid) FROM transactions WHERE id = $1 FOR UPDATE`, reversalOf).Scan(&originalReversal); err != nil || originalReversal != uuid.Nil {
			return fmt.Errorf("original transaction not found or is itself a reversal")
		}
		var replayID uuid.UUID
		replayErr := dbTx.QueryRow(ctx, `SELECT id FROM transactions WHERE idempotency_key = $1`, tx.IdempotencyKey).Scan(&replayID)
		if replayErr == nil {
			stored, err := r.GetTransaction(ctx, replayID)
			if err != nil {
				return err
			}
			if stored.SamePayload(tx) {
				return nil
			}
			return fmt.Errorf("%w: %s", ErrIdempotencyConflict, replayID)
		}
		if !errors.Is(replayErr, pgx.ErrNoRows) {
			return replayErr
		}
		var existing uuid.UUID
		err := dbTx.QueryRow(ctx, `SELECT id FROM transactions WHERE reversal_of = $1`, reversalOf).Scan(&existing)
		if err == nil {
			return fmt.Errorf("transaction already reversed")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	var inserted uuid.UUID
	err = dbTx.QueryRow(ctx, `
		INSERT INTO transactions (id, idempotency_key, description, status, reversal_of)
		VALUES ($1, $2, $3, $4, NULLIF($5, '00000000-0000-0000-0000-000000000000'::uuid))
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`, tx.ID, tx.IdempotencyKey, tx.Description, tx.Status, reversalOf).Scan(&inserted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var existing uuid.UUID
			if qerr := dbTx.QueryRow(ctx, `
				SELECT id FROM transactions WHERE idempotency_key = $1
			`, tx.IdempotencyKey).Scan(&existing); qerr != nil {
				return qerr
			}
			stored, qerr := r.GetTransaction(ctx, existing)
			if qerr != nil {
				return qerr
			}
			if stored.SamePayload(tx) {
				return nil
			}
			return fmt.Errorf("%w: %s", ErrIdempotencyConflict, existing)
		}
		return err
	}

	usedAccounts := make(map[domain.AccountID]struct{})
	for _, entry := range tx.Entries {
		usedAccounts[entry.AccountID] = struct{}{}
	}
	accountIDs := make([]domain.AccountID, 0, len(usedAccounts))
	for id := range usedAccounts {
		accountIDs = append(accountIDs, id)
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i].String() < accountIDs[j].String() })
	for _, id := range accountIDs {
		accountCurrency, status := "", ""
		var version int64
		if err := dbTx.QueryRow(ctx, `SELECT currency, status, version FROM accounts WHERE id = $1 FOR UPDATE`, id).Scan(&accountCurrency, &status, &version); err != nil {
			return fmt.Errorf("account not found: %s: %w", id, err)
		}
		if status != string(domain.AccountActive) {
			return fmt.Errorf("account is inactive: %s", id)
		}
		if expected, ok := tx.ExpectedVersions[id]; ok && version != expected {
			return fmt.Errorf("%w: account %s expected %d actual %d", ErrVersionConflict, id, expected, version)
		}
		for _, entry := range tx.Entries {
			if entry.AccountID == id && entry.Currency != accountCurrency {
				return fmt.Errorf("account currency does not match: %s", id)
			}
		}
	}
	for _, e := range tx.Entries {
		if _, err := dbTx.Exec(ctx, `INSERT INTO entries (id, transaction_id, account_id, amount, direction, currency) VALUES ($1, $2, $3, $4, $5, $6)`, e.ID, e.TransactionID, e.AccountID, e.Amount.String(), string(e.Direction), e.Currency); err != nil {
			return err
		}
	}
	for _, id := range accountIDs {
		if _, err := dbTx.Exec(ctx, `UPDATE accounts SET version = version + 1 WHERE id = $1`, id); err != nil {
			return err
		}
	}

	return dbTx.Commit(ctx)
}

func (r *PostgresLedger) GetTransaction(ctx context.Context, id domain.TransactionID) (domain.Transaction, error) {
	return r.getTransaction(ctx, `SELECT id, idempotency_key, COALESCE(description, ''), status, COALESCE(reversal_of, '00000000-0000-0000-0000-000000000000'::uuid) FROM transactions WHERE id = $1`, id)
}

func (r *PostgresLedger) GetTransactionByIdempotencyKey(ctx context.Context, key string) (domain.Transaction, error) {
	return r.getTransaction(ctx, `SELECT id, idempotency_key, COALESCE(description, ''), status, COALESCE(reversal_of, '00000000-0000-0000-0000-000000000000'::uuid) FROM transactions WHERE idempotency_key = $1`, key)
}

func (r *PostgresLedger) getTransaction(ctx context.Context, sql string, arg any) (domain.Transaction, error) {
	var tx domain.Transaction
	if err := r.pool.QueryRow(ctx, sql, arg).Scan(&tx.ID, &tx.IdempotencyKey, &tx.Description, &tx.Status, &tx.ReversalOf); err != nil {
		return domain.Transaction{}, err
	}
	entries, err := r.pool.Query(ctx, `SELECT id, transaction_id, account_id, amount::text, direction, currency, created_at FROM entries WHERE transaction_id = $1 ORDER BY id`, tx.ID)
	if err != nil {
		return domain.Transaction{}, err
	}
	defer entries.Close()
	for entries.Next() {
		var entry domain.Entry
		var amount, direction string
		if err := entries.Scan(&entry.ID, &entry.TransactionID, &entry.AccountID, &amount, &direction, &entry.Currency, &entry.CreatedAt); err != nil {
			return domain.Transaction{}, err
		}
		entry.Amount, err = decimal.NewFromString(amount)
		if err != nil {
			return domain.Transaction{}, err
		}
		entry.Direction = domain.Direction(direction)
		tx.Entries = append(tx.Entries, entry)
	}
	return tx, entries.Err()
}

func (r *PostgresLedger) GetEntriesPage(ctx context.Context, accountID domain.AccountID, limit, offset int) ([]domain.Entry, error) {
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	rows, err := r.pool.Query(ctx, `SELECT id, transaction_id, account_id, amount::text, direction, currency, created_at FROM entries WHERE account_id = $1 ORDER BY created_at, id LIMIT $2 OFFSET $3`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (r *PostgresLedger) GetEntriesAfter(ctx context.Context, accountID domain.AccountID, limit int, createdAt time.Time, entryID uuid.UUID) ([]domain.Entry, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	rows, err := r.pool.Query(ctx, `SELECT id, transaction_id, account_id, amount::text, direction, currency, created_at FROM entries WHERE account_id = $1 AND (created_at, id) > ($2, $3) ORDER BY created_at, id LIMIT $4`, accountID, createdAt, entryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (r *PostgresLedger) GetBalanceAsOf(ctx context.Context, accountID domain.AccountID, asOf time.Time) (decimal.Decimal, string, error) {
	var amountText, currency string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN e.direction = 'DEBIT' THEN e.amount ELSE -e.amount END), 0)::text, a.currency FROM accounts a LEFT JOIN entries e ON e.account_id = a.id AND e.created_at <= $2 WHERE a.id = $1 GROUP BY a.currency`, accountID, asOf).Scan(&amountText, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, "USD", nil
	}
	if err != nil {
		return decimal.Zero, "", err
	}
	balance, err := decimal.NewFromString(amountText)
	return balance, currency, err
}

func (r *PostgresLedger) GetTrialBalance(ctx context.Context) ([]AccountBalance, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id, a.currency, COALESCE(SUM(CASE WHEN e.direction = 'DEBIT' THEN e.amount ELSE -e.amount END), 0)::text FROM accounts a LEFT JOIN entries e ON e.account_id = a.id GROUP BY a.id, a.currency ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountBalance
	for rows.Next() {
		var row AccountBalance
		var amount string
		if err := rows.Scan(&row.AccountID, &row.Currency, &amount); err != nil {
			return nil, err
		}
		row.Balance, err = decimal.NewFromString(amount)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PostgresLedger) GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error) {
	var amountText string
	var currency *string
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN e.direction = 'DEBIT' THEN e.amount ELSE -e.amount END), 0)::text, a.currency
		FROM accounts a LEFT JOIN entries e ON e.account_id = a.id
		WHERE a.id = $1 GROUP BY a.currency
	`, accountID).Scan(&amountText, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, "USD", nil
	}
	if err != nil {
		return decimal.Zero, "", err
	}
	bal, err := decimal.NewFromString(amountText)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("parse balance: %w", err)
	}
	cur := "USD"
	if currency != nil && *currency != "" {
		cur = *currency
	}
	return bal, cur, nil
}

func scanEntries(rows pgx.Rows) ([]domain.Entry, error) {
	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		var amountText, direction string
		if err := rows.Scan(&e.ID, &e.TransactionID, &e.AccountID, &amountText, &direction, &e.Currency, &e.CreatedAt); err != nil {
			return nil, err
		}
		amount, err := decimal.NewFromString(amountText)
		if err != nil {
			return nil, fmt.Errorf("parse amount: %w", err)
		}
		e.Amount, e.Direction = amount, domain.Direction(direction)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresLedger) GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, transaction_id, account_id, amount::text, direction, currency, created_at
		FROM entries
		WHERE account_id = $1
		ORDER BY created_at ASC, id ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		var amountText, direction string
		if err := rows.Scan(&e.ID, &e.TransactionID, &e.AccountID, &amountText, &direction, &e.Currency, &e.CreatedAt); err != nil {
			return nil, err
		}
		amt, err := decimal.NewFromString(amountText)
		if err != nil {
			return nil, fmt.Errorf("parse amount: %w", err)
		}
		e.Amount = amt
		e.Direction = domain.Direction(direction)
		out = append(out, e)
	}
	return out, rows.Err()
}

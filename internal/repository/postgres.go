package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Compile-time checks that both stores satisfy the command/query port.
var (
	_ LedgerRepository = (*PostgresLedger)(nil)
	_ LedgerRepository = (*InMemoryLedger)(nil)
)

// PostgresLedger persists transactions against migrations/001_init.sql.
// Accounts referenced by entries are inserted if missing so the existing
// post-transaction API can run without a separate create-account call.
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

func (r *PostgresLedger) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	var inserted uuid.UUID
	err = dbTx.QueryRow(ctx, `
		INSERT INTO transactions (id, idempotency_key, description, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`, tx.ID, tx.IdempotencyKey, tx.Description, tx.Status).Scan(&inserted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var existing uuid.UUID
			if qerr := dbTx.QueryRow(ctx, `
				SELECT id FROM transactions WHERE idempotency_key = $1
			`, tx.IdempotencyKey).Scan(&existing); qerr != nil {
				return qerr
			}
			return fmt.Errorf("idempotency key already used: %s", existing)
		}
		return err
	}

	for _, e := range tx.Entries {
		if _, err := dbTx.Exec(ctx, `
			INSERT INTO accounts (id, owner_id, currency, type, status)
			VALUES ($1, $2, $3, 'ASSET', 'ACTIVE')
			ON CONFLICT (id) DO NOTHING
		`, e.AccountID, "unspecified", e.Currency); err != nil {
			return err
		}

		if _, err := dbTx.Exec(ctx, `
			INSERT INTO entries (id, transaction_id, account_id, amount, direction, currency)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, e.ID, e.TransactionID, e.AccountID, e.Amount.String(), string(e.Direction), e.Currency); err != nil {
			return err
		}
	}

	return dbTx.Commit(ctx)
}

func (r *PostgresLedger) GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error) {
	var amountText string
	var currency *string
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN direction = 'DEBIT' THEN amount ELSE -amount END), 0)::text,
			(
				SELECT currency FROM entries
				WHERE account_id = $1
				ORDER BY created_at DESC, id DESC
				LIMIT 1
			)
		FROM entries
		WHERE account_id = $1
	`, accountID).Scan(&amountText, &currency)
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

func (r *PostgresLedger) GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, transaction_id, account_id, amount::text, direction, currency
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
		if err := rows.Scan(&e.ID, &e.TransactionID, &e.AccountID, &amountText, &direction, &e.Currency); err != nil {
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

package command

import (
	"context"
	"fmt"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostTransactionCommand struct {
	IdempotencyKey string
	Description    string
	Entries        []EntryInput
}

type EntryInput struct {
	AccountID domain.AccountID
	Amount    decimal.Decimal
	Direction domain.Direction
	Currency  string
}

type PostTransactionHandler struct {
	repo repository.LedgerRepository
}

func NewPostTransactionHandler(repo repository.LedgerRepository) *PostTransactionHandler {
	return &PostTransactionHandler{repo: repo}
}

func (h *PostTransactionHandler) Handle(ctx context.Context, cmd PostTransactionCommand) (domain.TransactionID, error) {
	tx := domain.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: cmd.IdempotencyKey,
		Description:    cmd.Description,
		Status:         "POSTED",
	}

	for _, e := range cmd.Entries {
		tx.Entries = append(tx.Entries, domain.Entry{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			AccountID:     e.AccountID,
			Amount:        e.Amount,
			Direction:     e.Direction,
			Currency:      e.Currency,
		})
	}

	if !tx.IsBalanced() {
		return uuid.Nil, fmt.Errorf("transaction is not balanced")
	}

	if err := h.repo.SaveTransaction(ctx, tx); err != nil {
		return uuid.Nil, err
	}
	return tx.ID, nil
}

package command

import (
	"context"
	"fmt"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
)

type ReverseTransactionHandler struct{ repo repository.LedgerRepository }

func NewReverseTransactionHandler(repo repository.LedgerRepository) *ReverseTransactionHandler {
	return &ReverseTransactionHandler{repo: repo}
}

func (h *ReverseTransactionHandler) Handle(ctx context.Context, originalID domain.TransactionID, idempotencyKey string) (domain.TransactionID, error) {
	original, err := h.repo.GetTransaction(ctx, originalID)
	if err != nil {
		return uuid.Nil, err
	}
	if original.ReversalOf != uuid.Nil {
		return uuid.Nil, fmt.Errorf("cannot reverse a reversal transaction")
	}
	if idempotencyKey == "" {
		return uuid.Nil, fmt.Errorf("idempotency key is required")
	}
	reversal := domain.Transaction{ID: uuid.New(), ReversalOf: originalID, IdempotencyKey: idempotencyKey, Description: "Reversal: " + original.Description, Status: "POSTED"}
	for _, entry := range original.Entries {
		direction := domain.Debit
		if entry.Direction == domain.Debit {
			direction = domain.Credit
		}
		reversal.Entries = append(reversal.Entries, domain.Entry{ID: uuid.New(), TransactionID: reversal.ID, AccountID: entry.AccountID, Amount: entry.Amount, Direction: direction, Currency: entry.Currency})
	}
	if err := h.repo.SaveReversal(ctx, originalID, reversal); err != nil {
		return uuid.Nil, err
	}
	stored, err := h.repo.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return uuid.Nil, err
	}
	return stored.ID, nil
}

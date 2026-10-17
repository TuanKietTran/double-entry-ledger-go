package query

import (
	"context"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

type GetTransactionHandler struct {
	repo repository.LedgerRepository
}

func NewGetTransactionHandler(repo repository.LedgerRepository) *GetTransactionHandler {
	return &GetTransactionHandler{repo: repo}
}

func (h *GetTransactionHandler) ByID(ctx context.Context, id domain.TransactionID) (domain.Transaction, error) {
	return h.repo.GetTransaction(ctx, id)
}

func (h *GetTransactionHandler) ByIdempotencyKey(ctx context.Context, key string) (domain.Transaction, error) {
	return h.repo.GetTransactionByIdempotencyKey(ctx, key)
}

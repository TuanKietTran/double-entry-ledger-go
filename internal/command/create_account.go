package command

import (
	"context"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

type CreateAccountHandler struct {
	repo repository.AccountRepository
}

func NewCreateAccountHandler(repo repository.AccountRepository) *CreateAccountHandler {
	return &CreateAccountHandler{repo: repo}
}

func (h *CreateAccountHandler) Handle(ctx context.Context, account domain.Account) error {
	if err := account.Validate(); err != nil {
		return err
	}
	return h.repo.CreateAccount(ctx, account)
}

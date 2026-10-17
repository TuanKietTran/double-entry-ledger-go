package command

import (
	"context"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
)

func TestCreateAccountHandlerValidationAndIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	h := NewCreateAccountHandler(repo)
	account := domain.Account{ID: uuid.New(), OwnerID: "fixture", Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}
	if err := h.Handle(ctx, account); err != nil {
		t.Fatal(err)
	}
	if err := h.Handle(ctx, account); err != nil {
		t.Fatalf("identical retry: %v", err)
	}
	bad := account
	bad.ID = uuid.Nil
	if err := h.Handle(ctx, bad); err == nil {
		t.Fatal("invalid account accepted")
	}
	changed := account
	changed.Currency = "EUR"
	if err := h.Handle(ctx, changed); err == nil {
		t.Fatal("changed immutable attributes accepted")
	}
}

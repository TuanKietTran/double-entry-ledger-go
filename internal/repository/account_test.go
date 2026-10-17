package repository

import (
	"context"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
)

func TestMemoryAccountProvisioningIsIdempotentAndImmutable(t *testing.T) {
	repo := NewInMemoryLedger()
	account := domain.Account{ID: uuid.New(), OwnerID: "fixture", Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}
	if err := repo.CreateAccount(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAccount(context.Background(), account); err != nil {
		t.Fatalf("same account retry: %v", err)
	}
	got, err := repo.GetAccount(context.Background(), account.ID)
	if err != nil || got != account {
		t.Fatalf("GetAccount() = %+v, %v; want %+v", got, err, account)
	}
	changed := account
	changed.Type = domain.Expense
	if err := repo.CreateAccount(context.Background(), changed); err == nil {
		t.Fatal("changed immutable account attributes unexpectedly accepted")
	}
}

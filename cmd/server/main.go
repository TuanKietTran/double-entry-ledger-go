package main

import (
	"context"
	"fmt"
	"log"

	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/example/double-entry-ledger/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// In-memory stub for demo
type memRepo struct{}

func (m *memRepo) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	fmt.Printf("Saved transaction %s with %d entries\n", tx.ID, len(tx.Entries))
	return nil
}
func (m *memRepo) GetBalance(ctx context.Context, id domain.AccountID) (decimal.Decimal, string, error) {
	return decimal.NewFromInt(1000), "USD", nil
}
func (m *memRepo) GetEntries(ctx context.Context, id domain.AccountID) ([]domain.Entry, error) {
	return nil, nil
}

func main() {
	repo := &memRepo{}
	postHandler := command.NewPostTransactionHandler(repo)
	balanceHandler := query.NewGetBalanceHandler(repo)

	accountID := uuid.New()
	cmd := command.PostTransactionCommand{
		IdempotencyKey: "demo-1",
		Description:    "Test transfer",
		Entries: []command.EntryInput{
			{AccountID: accountID, Amount: decimal.NewFromInt(100), Direction: domain.Debit, Currency: "USD"},
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(100), Direction: domain.Credit, Currency: "USD"},
		},
	}

	id, err := postHandler.Handle(context.Background(), cmd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Posted transaction:", id)

	bal, err := balanceHandler.Handle(context.Background(), query.GetBalanceQuery{AccountID: accountID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Balance: %s %s\n", bal.Available, bal.Currency)
}

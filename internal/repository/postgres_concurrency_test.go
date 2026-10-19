package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func testPostgresIndependentConnections(t *testing.T, first *PostgresLedger, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	second, err := NewPostgresLedger(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	var firstPID, secondPID int
	if err := first.pool.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&firstPID); err != nil {
		t.Fatal(err)
	}
	if err := second.pool.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&secondPID); err != nil {
		t.Fatal(err)
	}
	if firstPID == secondPID {
		t.Fatal("expected independent PostgreSQL connections")
	}
	stores := []*PostgresLedger{first, second}

	raceWrites := func(writes [2]func() error) [2]error {
		start := make(chan struct{})
		var ready, done sync.WaitGroup
		var results [2]error
		ready.Add(2)
		done.Add(2)
		for i := range writes {
			go func(i int) {
				defer done.Done()
				ready.Done()
				<-start
				results[i] = writes[i]()
			}(i)
		}
		ready.Wait()
		close(start)
		done.Wait()
		return results
	}
	newAccounts := func(t *testing.T) (uuid.UUID, uuid.UUID) {
		t.Helper()
		a, b := uuid.New(), uuid.New()
		for _, id := range []uuid.UUID{a, b} {
			if err := first.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
				t.Fatal(err)
			}
		}
		return a, b
	}
	makeTx := func(a, b uuid.UUID, key string, amount int64) domain.Transaction {
		id := uuid.New()
		return domain.Transaction{ID: id, IdempotencyKey: key, Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), TransactionID: id, AccountID: a, Amount: decimal.NewFromInt(amount), Direction: domain.Debit, Currency: "USD"},
			{ID: uuid.New(), TransactionID: id, AccountID: b, Amount: decimal.NewFromInt(amount), Direction: domain.Credit, Currency: "USD"},
		}}
	}
	assertState := func(t *testing.T, a, b uuid.UUID, amount int64, version int64, entryCount int) {
		t.Helper()
		for i, id := range []uuid.UUID{a, b} {
			want := decimal.NewFromInt(amount)
			if i == 1 {
				want = want.Neg()
			}
			balance, currency, err := second.GetBalance(ctx, id)
			if err != nil || currency != "USD" || !balance.Equal(want) {
				t.Fatalf("balance=%s %s err=%v; want %s USD", balance, currency, err, want)
			}
			account, err := second.GetAccount(ctx, id)
			if err != nil || account.Version != version {
				t.Fatalf("version=%d err=%v; want %d", account.Version, err, version)
			}
			entries, err := second.GetEntriesPage(ctx, id, 500, 0)
			if err != nil || len(entries) != entryCount {
				t.Fatalf("entries=%d err=%v; want %d", len(entries), err, entryCount)
			}
			fold := decimal.Zero
			for _, entry := range entries {
				if entry.Direction == domain.Debit {
					fold = fold.Add(entry.Amount)
				} else {
					fold = fold.Sub(entry.Amount)
				}
			}
			if !fold.Equal(balance) {
				t.Fatalf("entry fold=%s balance=%s", fold, balance)
			}
		}
	}

	t.Run("same payload replays once", func(t *testing.T) {
		a, b := newAccounts(t)
		key := uuid.NewString()
		txs := [2]domain.Transaction{makeTx(a, b, key, 7), makeTx(a, b, key, 7)}
		results := raceWrites([2]func() error{
			func() error { return stores[0].SaveTransaction(ctx, txs[0]) },
			func() error { return stores[1].SaveTransaction(ctx, txs[1]) },
		})
		for _, err := range results {
			if err != nil {
				t.Fatal(err)
			}
		}
		original, err := first.GetTransactionByIdempotencyKey(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		other, err := second.GetTransactionByIdempotencyKey(ctx, key)
		if err != nil || other.ID != original.ID {
			t.Fatalf("replay IDs differ: %v", err)
		}
		if original.ID != txs[0].ID && original.ID != txs[1].ID {
			t.Fatal("stored unexpected transaction identity")
		}
		assertState(t, a, b, 7, 1, 1)
	})

	for _, mode := range []string{"payload conflict", "stale version"} {
		t.Run(mode, func(t *testing.T) {
			a, b := newAccounts(t)
			key := uuid.NewString()
			txs := [2]domain.Transaction{makeTx(a, b, key, 7), makeTx(a, b, key, 8)}
			wantErr := ErrIdempotencyConflict
			if mode == "stale version" {
				txs[1].IdempotencyKey = uuid.NewString()
				for i := range txs {
					txs[i].ExpectedVersions = map[domain.AccountID]int64{a: 0, b: 0}
				}
				wantErr = ErrVersionConflict
			}
			results := raceWrites([2]func() error{
				func() error { return stores[0].SaveTransaction(ctx, txs[0]) },
				func() error { return stores[1].SaveTransaction(ctx, txs[1]) },
			})
			winner := -1
			for i, err := range results {
				if err == nil {
					if winner != -1 {
						t.Fatal("both conflicting writes committed")
					}
					winner = i
				} else if !errors.Is(err, wantErr) {
					t.Fatalf("unexpected conflict: %v; want %v", err, wantErr)
				}
			}
			if winner == -1 {
				t.Fatalf("neither committed: %v", results)
			}
			if _, err := first.GetTransaction(ctx, txs[1-winner].ID); err == nil {
				t.Fatal("losing transaction survived rollback")
			}
			assertState(t, a, b, int64(7+winner), 1, 1)
		})
	}

	t.Run("one reversal per original", func(t *testing.T) {
		a, b := newAccounts(t)
		original := makeTx(a, b, uuid.NewString(), 9)
		if err := first.SaveTransaction(ctx, original); err != nil {
			t.Fatal(err)
		}
		reversals := [2]domain.Transaction{makeTx(b, a, uuid.NewString(), 9), makeTx(b, a, uuid.NewString(), 9)}
		for i := range reversals {
			reversals[i].ReversalOf = original.ID
		}
		results := raceWrites([2]func() error{
			func() error { return first.SaveReversal(ctx, original.ID, reversals[0]) },
			func() error { return second.SaveReversal(ctx, original.ID, reversals[1]) },
		})
		winner := -1
		for i, err := range results {
			if err == nil {
				if winner != -1 {
					t.Fatal("both reversals committed")
				}
				winner = i
			} else if err.Error() != "transaction already reversed" {
				t.Fatalf("unexpected reversal error: %v", err)
			}
		}
		if winner == -1 {
			t.Fatalf("neither reversal committed: %v", results)
		}
		if _, err := first.GetTransaction(ctx, reversals[1-winner].ID); err == nil {
			t.Fatal("losing reversal survived rollback")
		}
		assertState(t, a, b, 0, 2, 2)
	})
}

func testPostgresTimestampTiePagination(t *testing.T, repo *PostgresLedger) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: uuid.NewString(), Status: "POSTED"}
	// PostgreSQL's transaction timestamp is identical for all these inserted legs.
	// Reverse UUID insertion order makes the deterministic tie-break observable.
	var want []uuid.UUID
	for i := 5; i >= 1; i-- {
		id := uuid.MustParse(fmt.Sprintf("10000000-0000-4000-8000-%012d", i))
		tx.Entries = append(tx.Entries, domain.Entry{ID: id, TransactionID: tx.ID, AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"})
		want = append([]uuid.UUID{id}, want...)
	}
	tx.Entries = append(tx.Entries, domain.Entry{ID: uuid.New(), TransactionID: tx.ID, AccountID: b, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "USD"})
	if err := repo.SaveTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	all, err := repo.GetEntriesPage(ctx, a, 100, 0)
	if err != nil || len(all) != 5 {
		t.Fatalf("all=%d err=%v", len(all), err)
	}
	var cursorPaged []domain.Entry
	cursorAt, cursorID := time.Time{}, uuid.Nil
	for {
		page, err := repo.GetEntriesAfter(ctx, a, 2, cursorAt, cursorID)
		if err != nil {
			t.Fatal(err)
		}
		cursorPaged = append(cursorPaged, page...)
		if len(page) < 2 {
			break
		}
		cursorAt, cursorID = page[len(page)-1].CreatedAt, page[len(page)-1].ID
	}
	if len(cursorPaged) != len(want) {
		t.Fatalf("cursor paged %d; want %d", len(cursorPaged), len(want))
	}
	var paged []domain.Entry
	for offset := 0; offset < 6; offset += 2 {
		page, err := repo.GetEntriesPage(ctx, a, 2, offset)
		if err != nil {
			t.Fatal(err)
		}
		paged = append(paged, page...)
	}
	if len(paged) != len(want) {
		t.Fatalf("paged %d; want %d", len(paged), len(want))
	}
	for i, id := range want {
		if all[i].ID != id || paged[i].ID != id || cursorPaged[i].ID != id {
			t.Fatalf("unexpected tie order at %d", i)
		}
		if !all[i].CreatedAt.Equal(all[0].CreatedAt) {
			t.Fatal("fixture timestamps are not tied")
		}
	}
	page, err := repo.GetEntriesPage(ctx, a, 2, 99)
	if err != nil || len(page) != 0 {
		t.Fatalf("past-end page=%d err=%v", len(page), err)
	}
}

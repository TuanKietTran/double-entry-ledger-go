package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCursorPaginationContinuesAcrossConcurrentAppend(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	post := command.NewPostTransactionHandler(repo)
	postOne := func(key string) {
		t.Helper()
		if _, err := post.Handle(ctx, command.PostTransactionCommand{IdempotencyKey: key, Entries: []command.EntryInput{{AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"}, {AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"}}}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		postOne(fmt.Sprintf("cursor-%d", i))
	}
	h := NewHandlersWithTransactionQueries(nil, query.NewGetBalanceHandler(repo), nil, nil, query.NewLedgerQueries(repo)).Routes()
	fetch := func(cursor string) (*httptest.ResponseRecorder, []struct {
		EntryID string `json:"entry_id"`
	}) {
		path := "/accounts/" + a.String() + "/entries?limit=1"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		var entries []struct {
			EntryID string `json:"entry_id"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &entries); err != nil {
			t.Fatal(err)
		}
		return res, entries
	}
	first, entries := fetch("")
	if first.Code != 200 || len(entries) != 1 || first.Header().Get("X-Next-Cursor") == "" {
		t.Fatalf("first page status=%d entries=%d headers=%v", first.Code, len(entries), first.Header())
	}
	seen := map[string]bool{entries[0].EntryID: true}
	cursor := first.Header().Get("X-Next-Cursor")
	postOne("cursor-appended-after-first-page")
	for cursor != "" {
		res, page := fetch(cursor)
		if res.Code != 200 {
			t.Fatalf("cursor page status=%d body=%s", res.Code, res.Body)
		}
		for _, entry := range page {
			if seen[entry.EntryID] {
				t.Fatalf("cursor repeated entry %s", entry.EntryID)
			}
			seen[entry.EntryID] = true
		}
		cursor = res.Header().Get("X-Next-Cursor")
	}
	if len(seen) != 4 {
		t.Fatalf("cursor walk returned %d unique entries, want 4", len(seen))
	}
}

func TestLedgerQueryRoutes(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := command.NewPostTransactionHandler(repo).Handle(ctx, command.PostTransactionCommand{IdempotencyKey: "query-http", Entries: []command.EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(2), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(2), Direction: domain.Credit, Currency: "USD"},
	}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandlersWithTransactionQueries(nil, query.NewGetBalanceHandler(repo), nil, nil, query.NewLedgerQueries(repo))
	at := time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)
	for _, path := range []string{"/accounts/" + a.String() + "/entries?limit=1&offset=0", "/accounts/" + a.String() + "/balance?as_of=" + at, "/trial-balance"} {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 200 {
			t.Errorf("GET %s = %d %s", path, res.Code, res.Body.String())
		}
	}
}

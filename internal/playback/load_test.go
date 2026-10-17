package playback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/example/double-entry-ledger/internal/api"
	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/example/double-entry-ledger/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func failFirst(next http.Handler) http.Handler {
	var mu sync.Mutex
	seen := map[string]int{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		mu.Lock()
		seen[string(body)]++
		n := seen[string(body)]
		mu.Unlock()
		if n == 1 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TestThousandPerMinuteAndPlayback(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	h := api.NewHandlers(command.NewPostTransactionHandler(repo), query.NewGetBalanceHandler(repo))
	router := chi.NewRouter()
	router.Use(failFirst)
	router.Mount("/v1", h.Routes())
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	cash := uuid.New()
	other := uuid.New()
	calls := make([]Call, RequestsPerMinute)
	for i := range calls {
		calls[i] = Call{
			Method: http.MethodPost,
			Path:   "/v1/transactions",
			Body: []byte(fmt.Sprintf(
				`{"idempotency_key":"k-%d","description":"load","entries":[{"account_id":"%s","amount":"1","direction":"DEBIT","currency":"USD"},{"account_id":"%s","amount":"1","direction":"CREDIT","currency":"USD"}]}`,
				i, cash, other)),
		}
	}

	start := time.Now()
	first, err := Play(context.Background(), srv.Client(), srv.URL, calls, DefaultPolicy())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed >= Window {
		t.Fatalf("handled %d requests in %s, want under %s", len(calls), elapsed, Window)
	}
	t.Logf("first play %d requests in %s", len(calls), elapsed)
	for i, o := range first {
		if o.Err != nil || o.Status != http.StatusOK || o.Attempts != 2 {
			t.Fatalf("call %d status %d attempts %d err %v body %s", i, o.Status, o.Attempts, o.Err, o.Body)
		}
	}
	assertBalance(t, srv, cash, decimal.NewFromInt(RequestsPerMinute))

	second, err := Play(context.Background(), srv.Client(), srv.URL, calls, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for i, o := range second {
		if o.Attempts != 1 || !IsIdempotentReplay(o.Status, o.Body) {
			t.Fatalf("playback %d status %d attempts %d body %s", i, o.Status, o.Attempts, o.Body)
		}
	}
	assertBalance(t, srv, cash, decimal.NewFromInt(RequestsPerMinute))
}

func assertBalance(t *testing.T, srv *httptest.Server, account uuid.UUID, want decimal.Decimal) {
	t.Helper()
	out, err := Play(context.Background(), srv.Client(), srv.URL, []Call{{
		Method: http.MethodGet,
		Path:   "/v1/accounts/" + account.String() + "/balance",
	}}, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Status != http.StatusOK {
		t.Fatalf("balance status %d body %s", out[0].Status, out[0].Body)
	}
	var got struct {
		Available decimal.Decimal
	}
	if err := json.Unmarshal(out[0].Body, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Available.Equal(want) {
		t.Fatalf("balance %s, want %s", got.Available, want)
	}
}

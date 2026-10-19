package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/serverruntime"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestR1TenThousandProductionHTTPRequests(t *testing.T) {
	const total, duplicateCount = 10000, 200
	repo := repository.NewInMemoryLedger()
	debit, credit := uuid.New(), uuid.New()
	for _, a := range []domain.Account{{ID: debit, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}, {ID: credit, Currency: "USD", Type: domain.Equity, Status: domain.AccountActive}} {
		if err := repo.CreateAccount(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	production := serverruntime.BuildRouter(repo)
	var mu sync.Mutex
	faulted := make(map[string]bool)
	faultCounts := map[int]int{}
	faults := production
	faults = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/transactions" {
			b, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(strings.NewReader(string(b)))
			var req struct {
				IdempotencyKey string `json:"idempotency_key"`
			}
			_ = json.Unmarshal(b, &req)
			index, _ := strconv.Atoi(strings.TrimPrefix(req.IdempotencyKey, "r1-"))
			mu.Lock()
			inject := !faulted[req.IdempotencyKey]
			faulted[req.IdempotencyKey] = true
			mu.Unlock()
			if inject {
				if index < 100 {
					mu.Lock()
					faultCounts[0]++
					mu.Unlock()
					time.Sleep(1500 * time.Millisecond)
					return
				}
				switch index % 100 {
				case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
					mu.Lock()
					faultCounts[http.StatusServiceUnavailable]++
					mu.Unlock()
					http.Error(w, "injected", http.StatusServiceUnavailable)
					return
				case 10, 11:
					mu.Lock()
					faultCounts[http.StatusTooManyRequests]++
					mu.Unlock()
					http.Error(w, "injected", http.StatusTooManyRequests)
					return
				}
			}
		}
		production.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(faults)
	t.Cleanup(srv.Close)
	calls := make([]Call, total)
	for i := range calls {
		keyIndex := i
		if i >= total-duplicateCount {
			keyIndex = i - (total - duplicateCount)
		}
		calls[i] = Call{Method: http.MethodPost, Path: "/v1/transactions", Body: []byte(fmt.Sprintf(`{"idempotency_key":"r1-%d","description":"synthetic replay","entries":[{"account_id":"%s","amount":"1","direction":"DEBIT","currency":"USD"},{"account_id":"%s","amount":"1","direction":"CREDIT","currency":"USD"}]}`, keyIndex, debit, credit))}
	}
	client := &http.Client{Transport: &http.Transport{MaxConnsPerHost: 128, MaxIdleConns: 128, MaxIdleConnsPerHost: 128}}
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), Window)
	defer cancel()
	start := time.Now()
	policy := DefaultPolicy()
	policy.AttemptTimeout = time.Second
	out, err := Play(ctx, client, srv.URL, calls, policy)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed >= Window {
		t.Fatalf("R1 elapsed %s exceeds %s", elapsed, Window)
	}
	mu.Lock()
	if faultCounts[http.StatusServiceUnavailable] < 500 || faultCounts[http.StatusTooManyRequests] == 0 || faultCounts[0] < 100 {
		t.Fatalf("injected faults: %+v", faultCounts)
	}
	mu.Unlock()
	for i, outcome := range out {
		if outcome.Err != nil || outcome.Status != http.StatusOK || outcome.Attempts < 1 || outcome.Attempts > 6 {
			t.Fatalf("call %d: status=%d attempts=%d err=%v body=%s", i, outcome.Status, outcome.Attempts, outcome.Err, outcome.Body)
		}
	}
	var original, duplicate struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := json.Unmarshal(out[0].Body, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out[total-duplicateCount].Body, &duplicate); err != nil {
		t.Fatal(err)
	}
	if original.TransactionID == "" || original.TransactionID != duplicate.TransactionID {
		t.Fatalf("same-key submissions returned different transaction IDs: %+v %+v", original, duplicate)
	}
	bal, _, err := repo.GetBalance(context.Background(), debit)
	if err != nil {
		t.Fatal(err)
	}
	if !bal.Equal(decimal.NewFromInt(total - duplicateCount)) {
		t.Fatalf("debit balance %s", bal)
	}
	account, err := repo.GetAccount(context.Background(), debit)
	if err != nil {
		t.Fatal(err)
	}
	if account.Version != total-duplicateCount {
		t.Fatalf("debit version %d, want %d", account.Version, total-duplicateCount)
	}
	rows, err := repo.GetTrialBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("trial balance rows %d, want 2", len(rows))
	}
	totalBalance := decimal.Zero
	for _, row := range rows {
		totalBalance = totalBalance.Add(row.Balance)
	}
	if !totalBalance.IsZero() {
		t.Fatalf("trial balance does not net to zero: %s", totalBalance)
	}
	t.Logf("R1 %d requests, %d duplicate submissions, %s elapsed, %d unique effects; 6 attempts max", total, duplicateCount, elapsed, total-duplicateCount)
}

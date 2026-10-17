package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

func TestAccountRouteValidationAndNotFound(t *testing.T) {
	mem := repository.NewInMemoryLedger()
	h := NewHandlersWithLedger(nil, nil, command.NewCreateAccountHandler(mem), nil, query.NewLedgerQueries(mem), nil)
	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/accounts/not-a-uuid", want: 400},
		{path: "/accounts/550e8400-e29b-41d4-a716-446655440000", want: 404},
	} {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest("GET", tc.path, nil))
		if res.Code != tc.want {
			t.Errorf("GET %s status=%d want=%d body=%s", tc.path, res.Code, tc.want, res.Body.String())
		}
	}
}

func TestAccountQueryRejectsInvalidPaginationAndTimestamp(t *testing.T) {
	mem := repository.NewInMemoryLedger()
	h := NewHandlersWithLedger(nil, query.NewGetBalanceHandler(mem), nil, nil, query.NewLedgerQueries(mem), nil)
	id := "550e8400-e29b-41d4-a716-446655440000"
	for _, path := range []string{"/accounts/" + id + "/entries?limit=0", "/accounts/" + id + "/entries?limit=501", "/accounts/" + id + "/entries?limit=abc", "/accounts/" + id + "/balance?as_of=not-time"} {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 400 || strings.TrimSpace(res.Body.String()) == "" {
			t.Errorf("GET %s = %d %q, want a 400 error", path, res.Code, res.Body.String())
		}
	}
}

func TestAccountEntriesRejectsInvalidOffset(t *testing.T) {
	mem := repository.NewInMemoryLedger()
	h := NewHandlersWithLedger(nil, nil, nil, nil, query.NewLedgerQueries(mem), nil)
	id := "550e8400-e29b-41d4-a716-446655440000"
	for _, path := range []string{"/accounts/" + id + "/entries?offset=-1", "/accounts/" + id + "/entries?offset=abc", "/accounts/not-a-uuid/entries"} {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 400 || strings.TrimSpace(res.Body.String()) == "" {
			t.Errorf("GET %s = %d %q, want a 400 error", path, res.Code, res.Body.String())
		}
	}
}

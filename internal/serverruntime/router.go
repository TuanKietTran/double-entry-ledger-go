package serverruntime

import (
	"expvar"
	"net/http"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/api"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/go-chi/chi/v5"
)

var requestCount = expvar.NewInt("ledger_http_requests_total")

// BuildRouter constructs the same production router used by cmd/server.
func BuildRouter(repo repository.LedgerRepository) http.Handler {
	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	accounts, ok := repo.(repository.AccountRepository)
	if !ok {
		panic("ledger repository does not support account provisioning")
	}
	apiHandlers := api.NewHandlersWithLedger(postH, balH, command.NewCreateAccountHandler(accounts), query.NewGetTransactionHandler(repo), query.NewLedgerQueries(repo), command.NewReverseTransactionHandler(repo))
	r := chi.NewRouter()
	r.Use(api.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { requestCount.Add(1); next.ServeHTTP(w, req) })
	})
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if repo == nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	r.Handle("/metrics", expvar.Handler())
	r.Mount("/v1", apiHandlers.Routes())
	return r
}

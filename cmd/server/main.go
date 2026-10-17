package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/api"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	repo, cleanup := openRepo(context.Background())
	defer cleanup()

	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	accountRepo, ok := repo.(repository.AccountRepository)
	if !ok {
		log.Fatal("ledger repository does not support account provisioning")
	}
	createAccountH := command.NewCreateAccountHandler(accountRepo)
	transactionQueryH := query.NewGetTransactionHandler(repo)
	ledgerQueriesH := query.NewLedgerQueries(repo)
	reverseH := command.NewReverseTransactionHandler(repo)
	handlers := api.NewHandlersWithLedger(postH, balH, createAccountH, transactionQueryH, ledgerQueriesH, reverseH)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/v1", handlers.Routes())

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8101"
	}
	log.Printf("Ledger API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func openRepo(ctx context.Context) (repository.LedgerRepository, func()) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("DATABASE_URL unset; using in-memory ledger")
		return repository.NewInMemoryLedger(), func() {}
	}
	repo, err := repository.NewPostgresLedger(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres ledger: %v", err)
	}
	log.Println("using postgres ledger")
	return repo, repo.Close
}

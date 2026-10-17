package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/example/double-entry-ledger/internal/api"
	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/example/double-entry-ledger/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	repo, cleanup := openRepo(context.Background())
	defer cleanup()

	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	handlers := api.NewHandlers(postH, balH)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/v1", handlers.Routes())

	log.Println("Ledger API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
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

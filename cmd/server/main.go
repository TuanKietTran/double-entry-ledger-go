package main

import (
	"log"
	"net/http"

	"github.com/example/double-entry-ledger/internal/api"
	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/example/double-entry-ledger/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	repo := repository.NewInMemoryLedger()
	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	handlers := api.NewHandlers(postH, balH)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/v1", handlers.Routes())

	log.Println("Ledger API (in-memory) listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

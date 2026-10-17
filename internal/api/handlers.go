package api

import (
	"encoding/json"
	"net/http"

	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handlers struct {
	postTx *command.PostTransactionHandler
	getBal *query.GetBalanceHandler
}

func NewHandlers(post *command.PostTransactionHandler, bal *query.GetBalanceHandler) *Handlers {
	return &Handlers{postTx: post, getBal: bal}
}

func (h *Handlers) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/transactions", h.postTransaction)
	r.Get("/accounts/{id}/balance", h.getBalance)
	return r
}

type postTxRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Description    string `json:"description"`
	Entries        []struct {
		AccountID string          `json:"account_id"`
		Amount    decimal.Decimal `json:"amount"`
		Direction string          `json:"direction"`
		Currency  string          `json:"currency"`
	} `json:"entries"`
}

func (h *Handlers) postTransaction(w http.ResponseWriter, r *http.Request) {
	var req postTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	entries := make([]command.EntryInput, len(req.Entries))
	for i, e := range req.Entries {
		id, err := uuid.Parse(e.AccountID)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}
		dir := domain.Debit
		if e.Direction == "CREDIT" {
			dir = domain.Credit
		}
		entries[i] = command.EntryInput{
			AccountID: id,
			Amount:    e.Amount,
			Direction: dir,
			Currency:  e.Currency,
		}
	}

	id, err := h.postTx.Handle(r.Context(), command.PostTransactionCommand{
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
		Entries:        entries,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"transaction_id": id.String()})
}

func (h *Handlers) getBalance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	res, err := h.getBal.Handle(r.Context(), query.GetBalanceQuery{AccountID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

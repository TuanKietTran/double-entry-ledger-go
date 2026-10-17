package api

import (
	"encoding/json"
	"net/http"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handlers struct {
	postTx         *command.PostTransactionHandler
	getBal         *query.GetBalanceHandler
	createAccount  *command.CreateAccountHandler
	getTransaction *query.GetTransactionHandler
}

func NewHandlers(post *command.PostTransactionHandler, bal *query.GetBalanceHandler, accounts ...*command.CreateAccountHandler) *Handlers {
	h := &Handlers{postTx: post, getBal: bal}
	if len(accounts) != 0 {
		h.createAccount = accounts[0]
	}
	return h
}

func NewHandlersWithTransactionQueries(post *command.PostTransactionHandler, bal *query.GetBalanceHandler, accounts *command.CreateAccountHandler, transactions *query.GetTransactionHandler) *Handlers {
	h := NewHandlers(post, bal, accounts)
	h.getTransaction = transactions
	return h
}

func (h *Handlers) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/transactions", h.postTransaction)
	r.Get("/accounts/{id}/balance", h.getBalance)
	if h.createAccount != nil {
		r.Post("/accounts", h.createAccountRoute)
	}
	if h.getTransaction != nil {
		r.Get("/transactions/{id}", h.getTransactionByID)
		r.Get("/transactions/by-idempotency-key/{key}", h.getTransactionByKey)
	}
	return r
}

type createAccountRequest struct {
	AccountID string `json:"account_id"`
	OwnerID   string `json:"owner_id"`
	Currency  string `json:"currency"`
	Type      string `json:"type"`
}

func (h *Handlers) createAccountRoute(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(req.AccountID)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}
	account := domain.Account{ID: id, OwnerID: req.OwnerID, Currency: req.Currency, Type: domain.AccountType(req.Type), Status: domain.AccountActive}
	if err := h.createAccount.Handle(r.Context(), account); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"account_id": id.String()})
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
		if e.Direction != string(domain.Debit) && e.Direction != string(domain.Credit) {
			http.Error(w, "invalid direction", http.StatusBadRequest)
			return
		}
		dir := domain.Direction(e.Direction)
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

type transactionResponse struct {
	TransactionID  string          `json:"transaction_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Description    string          `json:"description"`
	Status         string          `json:"status"`
	Entries        []entryResponse `json:"entries"`
}

type entryResponse struct {
	EntryID   string `json:"entry_id"`
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
	Direction string `json:"direction"`
	Currency  string `json:"currency"`
}

func (h *Handlers) writeTransaction(w http.ResponseWriter, r *http.Request, tx domain.Transaction, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	response := transactionResponse{TransactionID: tx.ID.String(), IdempotencyKey: tx.IdempotencyKey, Description: tx.Description, Status: tx.Status}
	for _, entry := range tx.Entries {
		response.Entries = append(response.Entries, entryResponse{EntryID: entry.ID.String(), AccountID: entry.AccountID.String(), Amount: entry.Amount.String(), Direction: string(entry.Direction), Currency: entry.Currency})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) getTransactionByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid transaction id", http.StatusBadRequest)
		return
	}
	tx, err := h.getTransaction.ByID(r.Context(), id)
	h.writeTransaction(w, r, tx, err)
}

func (h *Handlers) getTransactionByKey(w http.ResponseWriter, r *http.Request) {
	tx, err := h.getTransaction.ByIdempotencyKey(r.Context(), chi.URLParam(r, "key"))
	h.writeTransaction(w, r, tx, err)
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

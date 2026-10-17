package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handlers struct {
	postTx         *command.PostTransactionHandler
	getBal         *query.GetBalanceHandler
	createAccount  *command.CreateAccountHandler
	getTransaction *query.GetTransactionHandler
	ledgerQueries  *query.LedgerQueries
	reverseTx      *command.ReverseTransactionHandler
}

func NewHandlers(post *command.PostTransactionHandler, bal *query.GetBalanceHandler, accounts ...*command.CreateAccountHandler) *Handlers {
	h := &Handlers{postTx: post, getBal: bal}
	if len(accounts) != 0 {
		h.createAccount = accounts[0]
	}
	return h
}

func NewHandlersWithTransactionQueries(post *command.PostTransactionHandler, bal *query.GetBalanceHandler, accounts *command.CreateAccountHandler, transactions *query.GetTransactionHandler, ledgerQueries ...*query.LedgerQueries) *Handlers {
	h := NewHandlers(post, bal, accounts)
	h.getTransaction = transactions
	if len(ledgerQueries) > 0 {
		h.ledgerQueries = ledgerQueries[0]
	}
	return h
}

func (h *Handlers) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Post("/transactions", h.postTransaction)
	r.Get("/accounts/{id}/balance", h.getBalance)
	if h.createAccount != nil {
		r.Post("/accounts", h.createAccountRoute)
	}
	if h.getTransaction != nil {
		r.Get("/transactions/{id}", h.getTransactionByID)
		r.Get("/transactions/by-idempotency-key/{key}", h.getTransactionByKey)
	}
	if h.reverseTx != nil {
		r.Post("/transactions/{id}/reversals", h.reverseTransaction)
	}
	if h.ledgerQueries != nil {
		r.Get("/accounts/{id}", h.getAccount)
		r.Get("/accounts/{id}/entries", h.getAccountEntries)
		r.Get("/trial-balance", h.getTrialBalance)
	}
	return r
}

func NewHandlersWithLedger(post *command.PostTransactionHandler, bal *query.GetBalanceHandler, accounts *command.CreateAccountHandler, transactions *query.GetTransactionHandler, ledgerQueries *query.LedgerQueries, reverse *command.ReverseTransactionHandler) *Handlers {
	h := NewHandlersWithTransactionQueries(post, bal, accounts, transactions, ledgerQueries)
	h.reverseTx = reverse
	return h
}

type reversalRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handlers) reverseTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, "invalid transaction id", http.StatusBadRequest)
		return
	}
	var req reversalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, "invalid json", http.StatusBadRequest)
		return
	}
	reversalID, err := h.reverseTx.Handle(r.Context(), id, req.IdempotencyKey)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrIdempotencyConflict) || errors.Is(err, repository.ErrVersionConflict) {
			status = http.StatusConflict
		}
		writeError(w, r, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"transaction_id": reversalID.String()})
}

type createAccountRequest struct {
	AccountID string `json:"account_id"`
	OwnerID   string `json:"owner_id"`
	Currency  string `json:"currency"`
	Type      string `json:"type"`
}

func (h *Handlers) createAccountRoute(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, "invalid json", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(req.AccountID)
	if err != nil {
		writeError(w, r, "invalid account id", http.StatusBadRequest)
		return
	}
	account := domain.Account{ID: id, OwnerID: req.OwnerID, Currency: req.Currency, Type: domain.AccountType(req.Type), Status: domain.AccountActive}
	if err := h.createAccount.Handle(r.Context(), account); err != nil {
		writeError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"account_id": id.String()})
}

type postTxRequest struct {
	IdempotencyKey   string           `json:"idempotency_key"`
	ExpectedVersions map[string]int64 `json:"expected_versions,omitempty"`
	Description      string           `json:"description"`
	Entries          []struct {
		AccountID string          `json:"account_id"`
		Amount    decimal.Decimal `json:"amount"`
		Direction string          `json:"direction"`
		Currency  string          `json:"currency"`
	} `json:"entries"`
}

func (h *Handlers) postTransaction(w http.ResponseWriter, r *http.Request) {
	var req postTxRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, "invalid json", http.StatusBadRequest)
		return
	}

	expectedVersions := make(map[domain.AccountID]int64, len(req.ExpectedVersions))
	for rawID, version := range req.ExpectedVersions {
		id, err := uuid.Parse(rawID)
		if err != nil || version < 0 {
			writeError(w, r, "invalid expected_versions", http.StatusBadRequest)
			return
		}
		expectedVersions[id] = version
	}
	entries := make([]command.EntryInput, len(req.Entries))
	for i, e := range req.Entries {
		id, err := uuid.Parse(e.AccountID)
		if err != nil {
			writeError(w, r, "invalid account id", http.StatusBadRequest)
			return
		}
		if e.Direction != string(domain.Debit) && e.Direction != string(domain.Credit) {
			writeError(w, r, "invalid direction", http.StatusBadRequest)
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
		IdempotencyKey:   req.IdempotencyKey,
		ExpectedVersions: expectedVersions,
		Description:      req.Description,
		Entries:          entries,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrIdempotencyConflict) || errors.Is(err, repository.ErrVersionConflict) {
			status = http.StatusConflict
		}
		writeError(w, r, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"transaction_id": id.String()})
}

type transactionResponse struct {
	TransactionID  string          `json:"transaction_id"`
	ReversalOf     string          `json:"reversal_of,omitempty"`
	IdempotencyKey string          `json:"idempotency_key"`
	Description    string          `json:"description"`
	Status         string          `json:"status"`
	Entries        []entryResponse `json:"entries"`
}

type entryResponse struct {
	CreatedAt time.Time `json:"created_at"`
	EntryID   string    `json:"entry_id"`
	AccountID string    `json:"account_id"`
	Amount    string    `json:"amount"`
	Direction string    `json:"direction"`
	Currency  string    `json:"currency"`
}

func (h *Handlers) writeTransaction(w http.ResponseWriter, r *http.Request, tx domain.Transaction, err error) {
	if err != nil {
		writeError(w, r, err.Error(), http.StatusNotFound)
		return
	}
	response := transactionResponse{TransactionID: tx.ID.String(), IdempotencyKey: tx.IdempotencyKey, Description: tx.Description, Status: tx.Status}
	if tx.ReversalOf != uuid.Nil {
		response.ReversalOf = tx.ReversalOf.String()
	}
	for _, entry := range tx.Entries {
		response.Entries = append(response.Entries, entryResponse{CreatedAt: entry.CreatedAt, EntryID: entry.ID.String(), AccountID: entry.AccountID.String(), Amount: entry.Amount.String(), Direction: string(entry.Direction), Currency: entry.Currency})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) getTransactionByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, "invalid transaction id", http.StatusBadRequest)
		return
	}
	tx, err := h.getTransaction.ByID(r.Context(), id)
	h.writeTransaction(w, r, tx, err)
}

func (h *Handlers) getTransactionByKey(w http.ResponseWriter, r *http.Request) {
	tx, err := h.getTransaction.ByIdempotencyKey(r.Context(), chi.URLParam(r, "key"))
	h.writeTransaction(w, r, tx, err)
}

func (h *Handlers) getAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, "invalid account id", http.StatusBadRequest)
		return
	}
	account, err := h.ledgerQueries.Account(r.Context(), id)
	if err != nil {
		writeError(w, r, err.Error(), http.StatusNotFound)
		return
	}
	response := map[string]any{"account_id": account.ID.String(), "owner_id": account.OwnerID, "currency": account.Currency, "type": account.Type, "status": account.Status, "version": account.Version}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) getAccountEntries(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, "invalid account id", http.StatusBadRequest)
		return
	}
	limit, offset := 50, 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, r, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, r, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = parsed
	}
	if limit > 500 {
		writeError(w, r, "limit exceeds 500", http.StatusBadRequest)
		return
	}
	entries, err := h.ledgerQueries.Entries(r.Context(), id, limit, offset)
	if err != nil {
		writeError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	response := make([]entryResponse, 0, len(entries))
	for _, entry := range entries {
		response = append(response, entryResponse{CreatedAt: entry.CreatedAt, EntryID: entry.ID.String(), AccountID: entry.AccountID.String(), Amount: entry.Amount.String(), Direction: string(entry.Direction), Currency: entry.Currency})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) getTrialBalance(w http.ResponseWriter, r *http.Request) {
	rows, err := h.ledgerQueries.TrialBalance(r.Context())
	if err != nil {
		writeError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	type trialBalanceRow struct {
		AccountID string `json:"account_id"`
		Currency  string `json:"currency"`
		Balance   string `json:"balance"`
	}
	response := make([]trialBalanceRow, 0, len(rows))
	for _, row := range rows {
		response = append(response, trialBalanceRow{AccountID: row.AccountID.String(), Currency: row.Currency, Balance: row.Balance.String()})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) getBalance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, r, "invalid account id", http.StatusBadRequest)
		return
	}

	var res query.BalanceResult
	if raw := r.URL.Query().Get("as_of"); raw != "" {
		if h.ledgerQueries == nil {
			writeError(w, r, "as-of query unavailable", http.StatusInternalServerError)
			return
		}
		asOf, parseErr := time.Parse(time.RFC3339Nano, raw)
		if parseErr != nil {
			writeError(w, r, "invalid as_of timestamp", http.StatusBadRequest)
			return
		}
		res, err = h.ledgerQueries.BalanceAsOf(r.Context(), id, asOf)
	} else {
		res, err = h.getBal.Handle(r.Context(), query.GetBalanceQuery{AccountID: id})
	}
	if err != nil {
		writeError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

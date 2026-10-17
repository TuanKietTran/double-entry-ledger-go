package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/api"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type config struct {
	addr        string
	databaseURL string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{addr: strings.TrimSpace(getenv("HTTP_ADDR")), databaseURL: strings.TrimSpace(getenv("DATABASE_URL"))}
	if cfg.addr == "" {
		cfg.addr = ":8101"
	}
	if _, _, err := net.SplitHostPort(cfg.addr); err != nil {
		return config{}, fmt.Errorf("HTTP_ADDR must be host:port: %w", err)
	}
	if cfg.databaseURL != "" {
		u, err := url.Parse(cfg.databaseURL)
		if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
			return config{}, errors.New("DATABASE_URL must be a postgres URL with a host")
		}
	}
	return cfg, nil
}

var requestCount = expvar.NewInt("ledger_http_requests_total")

func buildRouter(repo repository.LedgerRepository) http.Handler {
	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	accounts, ok := repo.(repository.AccountRepository)
	if !ok {
		panic("ledger repository does not support account provisioning")
	}
	apiHandlers := api.NewHandlersWithLedger(postH, balH, command.NewCreateAccountHandler(accounts), query.NewGetTransactionHandler(repo), query.NewLedgerQueries(repo), command.NewReverseTransactionHandler(repo))
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
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

func openRepo(ctx context.Context, dsn string) (repository.LedgerRepository, func(), error) {
	if dsn == "" {
		return repository.NewInMemoryLedger(), func() {}, nil
	}
	repo, err := repository.NewPostgresLedger(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres ledger: %w", err)
	}
	return repo, repo.Close, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	repo, cleanup, err := openRepo(startupCtx, cfg.databaseURL)
	if err != nil {
		return err
	}
	defer cleanup()
	server := &http.Server{Addr: cfg.addr, Handler: buildRouter(repo), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.addr, err)
	}
	logger.Info("ledger server listening", "address", cfg.addr, "database_configured", cfg.databaseURL != "")
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

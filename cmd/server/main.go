package main

import (
	"context"
	"errors"
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

	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/serverruntime"
)

type config struct {
	addr        string
	databaseURL string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{addr: strings.TrimSpace(getenv("HTTP_ADDR")), databaseURL: strings.TrimSpace(getenv("DATABASE_URL"))}
	if cfg.addr == "" {
		cfg.addr = "127.0.0.1:8101"
	}
	host, _, err := net.SplitHostPort(cfg.addr)
	if err != nil {
		return config{}, fmt.Errorf("HTTP_ADDR must be host:port: %w", err)
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return config{}, errors.New("HTTP_ADDR must use a loopback host")
		}
	}
	if cfg.databaseURL != "" {
		u, err := url.Parse(cfg.databaseURL)
		if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
			return config{}, errors.New("DATABASE_URL must be a postgres URL with a host")
		}
	}
	return cfg, nil
}

func buildRouter(repo repository.LedgerRepository) http.Handler {
	return serverruntime.BuildRouter(repo)
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

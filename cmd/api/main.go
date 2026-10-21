package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momerefe/Kabine101/db"
	"github.com/momerefe/Kabine101/internal/auth"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/database"
	"github.com/momerefe/Kabine101/internal/httpapi"
	"github.com/momerefe/Kabine101/internal/mailer"
	"github.com/momerefe/Kabine101/internal/management"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pc, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return errors.New("invalid database configuration")
	}
	pc.MaxConns = 5
	pc.ConnConfig.ConnectTimeout = 3 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return errors.New("cannot initialize database pool")
	}
	defer pool.Close()
	queries := database.New(pool)
	sender, err := mailer.New(cfg)
	if err != nil {
		return err
	}
	authentication, err := auth.New(pool, cfg, sender)
	if err != nil {
		return err
	}
	ready := func(ctx context.Context) error {
		var version int
		var dirty bool
		if err := pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
			return err
		}
		if dirty || version != db.SchemaVersion {
			return errors.New("schema version mismatch")
		}
		_, err := queries.GetPlatformName(ctx)
		return err
	}
	srv := &http.Server{Addr: cfg.HTTPAddress, Handler: httpapi.Router(ready, authentication, management.New(pool, authentication)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	result := make(chan error, 1)
	go func() { result <- srv.ListenAndServe() }()
	log.Info("API started", "address", cfg.HTTPAddress, "environment", cfg.Environment)
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

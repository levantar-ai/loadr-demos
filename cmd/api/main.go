// Command api is a small storefront HTTP API used to demonstrate loadr in CI.
//
// It connects to Postgres, runs its migrations + seed on boot, and serves a
// handful of routes (products, orders, auth, plus deliberate CPU-bound and
// slow endpoints) so a load test has something interesting to exercise.
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

	"github.com/levantar-ai/loadr-demos/internal/cache"
	"github.com/levantar-ai/loadr-demos/internal/server"
	"github.com/levantar-ai/loadr-demos/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	addr := getenv("ADDR", ":8080")
	dsn := getenv("DATABASE_URL", "postgres://demo:demo@localhost:5432/storefront?sslmode=disable")
	redisURL := os.Getenv("REDIS_URL") // optional: enables the top-sellers cache

	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		logger.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	c := cache.New(ctx, redisURL)
	logger.Info("cache", "enabled", c.Enabled())

	if err := st.Migrate(ctx); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	if err := st.Seed(ctx); err != nil {
		logger.Error("seed failed", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(st, c, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

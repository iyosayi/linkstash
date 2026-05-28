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

	"github.com/iyosayi/linkstash/internal/config"
	"github.com/iyosayi/linkstash/internal/database"
	"github.com/iyosayi/linkstash/internal/link"
	"github.com/iyosayi/linkstash/internal/server"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	logger := slog.Default()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Pool.Close()
	store := link.NewStore(db.Pool)
	app := server.NewServer(store, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      app.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	logger.Info("database connected")
	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server failed", "error", err)
	case sig := <-shutdownCh:
		logger.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server shutdown failed", "error", err)
			return
		}

		logger.Info("server shutdown complete")
	}

}

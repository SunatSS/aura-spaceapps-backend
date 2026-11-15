// Command aura runs the AURA HTTP API server.
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

	"aura/internal/catalog"
	"aura/internal/config"
	"aura/internal/httpapi"
	"aura/internal/seed"
	"aura/internal/storage"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cat, err := catalog.Default()
	if err != nil {
		return err
	}

	db, err := storage.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.Seed {
		empty, err := db.Q().IsEmpty(ctx)
		if err != nil {
			return err
		}
		if empty {
			if err := seed.Run(ctx, db, cat, time.Now()); err != nil {
				return err
			}
			logger.Info("database was empty: demo mission seeded")
		}
		// A demo that stays up keeps its readings fresh (see seed.KeepFresh).
		go func() {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					if err := seed.KeepFresh(ctx, db, cat, time.Now()); err != nil && ctx.Err() == nil {
						logger.Warn("demo keep-fresh failed", "err", err)
					}
				}
			}
		}()
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(httpapi.Deps{DB: db, Catalog: cat, Logger: logger, StaticDir: cfg.StaticDir, Demo: cfg.Seed, IngestToken: cfg.IngestToken}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.HTTPAddr, "db", cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

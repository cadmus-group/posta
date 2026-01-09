package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/httpapi"
	"github.com/cadmus-group/posta/internal/queue"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	qc := queue.NewClient(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	defer func() { _ = qc.Close() }()

	app := httpapi.New(log, cfg, qc)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	go func() {
		log.Info("queue-api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
	log.Info("queue-api stopped")
}

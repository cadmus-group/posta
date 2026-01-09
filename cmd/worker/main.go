package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/queue"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	w := queue.NewWorker(log)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPass,
			DB:       cfg.RedisDB,
		},
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues: map[string]int{
				cfg.QueueCritical: cfg.WeightCritical,
				cfg.QueueDefault:  cfg.WeightDefault,
			},
		},
	)

	// Start in background so we can trap signals.
	errCh := make(chan error, 1)
	go func() {
		log.Info("worker started")
		errCh <- srv.Run(w.Mux())
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		log.Info("worker stopping...")
		// Shutdown waits for active tasks to finish (with a default timeout behavior).
		srv.Shutdown()
		log.Info("worker stopped")
	case err := <-errCh:
		// Run returns non-nil when server stops unexpectedly.
		log.Error("worker exited", "err", err)
		time.Sleep(250 * time.Millisecond) // give logger time in some runtimes
		os.Exit(1)
	}
}

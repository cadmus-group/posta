package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/modules"
	"github.com/cadmus-group/posta/internal/queue"
	"github.com/cadmus-group/posta/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Print(version.Info())
		return
	}

	cfg := config.LoadWithPrefix("WORKER")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	reg, err := modules.BuildRegistry(cfg.Modules, cfg.AllowedTaskTypes, log)
	if err != nil {
		log.Error("failed to load modules", "err", err)
		os.Exit(1)
	}

	mux, err := queue.NewServeMux(reg, nil)
	if err != nil {
		log.Error("failed to build worker mux", "err", err)
		os.Exit(1)
	}

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
		errCh <- srv.Run(mux)
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

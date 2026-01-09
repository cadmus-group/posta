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

	s := asynq.NewScheduler(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPass,
			DB:       cfg.RedisDB,
		},
		&asynq.SchedulerOpts{},
	)

	if err := queue.RegisterSchedules(s, cfg.QueueDefault); err != nil {
		log.Error("register schedules failed", "err", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("scheduler started")
		errCh <- s.Run()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		log.Info("scheduler stopping...")
		s.Shutdown()
		log.Info("scheduler stopped")
	case err := <-errCh:
		log.Error("scheduler exited", "err", err)
		time.Sleep(250 * time.Millisecond)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/httpapi"
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

	cfg := config.LoadWithPrefix("QUEUE_API")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	qc := queue.NewClient(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	defer func() { _ = qc.Close() }()

	reg, err := modules.BuildRegistry(cfg.Modules, cfg.AllowedTaskTypes, log)
	if err != nil {
		log.Error("failed to load modules", "err", err)
		os.Exit(1)
	}

	app := httpapi.New(log, cfg, qc, reg)

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

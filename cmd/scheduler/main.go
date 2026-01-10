package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/version"
	demomod "github.com/cadmus-group/posta/modules/demo"
)

func main() {
	versionFlag := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Print(version.Info())
		return
	}

	cfg := config.LoadWithPrefix("SCHEDULER")
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

	modules := cfg.Modules
	enableDemo := cfg.SchedulerEnableDemoJobs
	if len(cfg.AllowedTaskTypes) > 0 && !containsTask(cfg.AllowedTaskTypes, demomod.TypeExamplePing) {
		enableDemo = false
	}

	if !enableDemo {
		filtered := make([]string, 0, len(modules))
		for _, m := range modules {
			if strings.EqualFold(m, "demo") {
				continue
			}
			filtered = append(filtered, m)
		}
		modules = filtered
	}

	if err := modules.RegisterModuleSchedules(modules, log, s, cfg.QueueDefault); err != nil {
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

func containsTask(list []string, target string) bool {
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return true
		}
	}
	return false
}

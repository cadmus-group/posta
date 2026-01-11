package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cadmus-group/posta/internal/openapiagg"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	addr := getenv("HTTP_ADDR", ":8083")
	cfgPath := resolveConfigPath(logger)
	refreshOverride := parseRefreshOverride()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := openapiagg.NewServer(cfgPath, refreshOverride, nil, logger)

	logger.Info("starting openapi-aggregator", "addr", addr, "config", cfgPath, "refreshOverride", refreshOverride)
	if err := srv.Run(ctx, addr); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func resolveConfigPath(log *slog.Logger) string {
	if v := os.Getenv("OPENAPI_SOURCES_FILE"); v != "" {
		return v
	}
	if _, err := os.Stat(openapiagg.DefaultConfigPath()); err == nil {
		return openapiagg.DefaultConfigPath()
	}
	example := "config/openapi-sources.example.yaml"
	if _, err := os.Stat(example); err == nil {
		log.Warn("using example config", "path", example)
		return example
	}
	return openapiagg.DefaultConfigPath()
}

func parseRefreshOverride() time.Duration {
	if v := os.Getenv("OPENAPI_REFRESH_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 0
}

package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// queue-api
	HTTPAddr string

	// redis
	RedisAddr string
	RedisPass string
	RedisDB   int

	// asynq server
	Concurrency int

	// queues
	QueueDefault   string
	QueueCritical  string
	WeightDefault  int
	WeightCritical int

	// api safety
	MaxBodyBytes int64

	// optional: common timeouts
	ReadHeaderTimeout time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:  getEnv("HTTP_ADDR", ":8081"),
		RedisAddr: getEnv("REDIS_ADDR", "redis:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		Concurrency: getEnvInt("ASYNQ_CONCURRENCY", 20),

		QueueDefault:  getEnv("QUEUE_DEFAULT", "default"),
		QueueCritical: getEnv("QUEUE_CRITICAL", "critical"),

		WeightDefault:  getEnvInt("QUEUE_WEIGHT_DEFAULT", 5),
		WeightCritical: getEnvInt("QUEUE_WEIGHT_CRITICAL", 10),

		MaxBodyBytes:       int64(getEnvInt("HTTP_MAX_BODY_BYTES", 1_048_576)), // 1 MiB
		ReadHeaderTimeout:  time.Duration(getEnvInt("HTTP_READ_HEADER_TIMEOUT_MS", 5000)) * time.Millisecond,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

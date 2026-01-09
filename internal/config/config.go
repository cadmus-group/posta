package config

import (
	"os"
	"strconv"
	"strings"
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
	QueueAllowList []string

	// api safety
	MaxBodyBytes int64

	// optional: common timeouts
	ReadHeaderTimeout time.Duration

	// optional: enqueue auth
	EnqueueAuthToken string

	// redis client timeouts
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration
	RedisPingTimeout  time.Duration
}

func Load() Config {
	cfg := Config{
		HTTPAddr:  getEnv("HTTP_ADDR", ":8081"),
		RedisAddr: getEnv("REDIS_ADDR", "redis:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		Concurrency: getEnvInt("ASYNQ_CONCURRENCY", 20),

		QueueDefault:  getEnv("QUEUE_DEFAULT", "default"),
		QueueCritical: getEnv("QUEUE_CRITICAL", "critical"),

		WeightDefault:  getEnvInt("QUEUE_WEIGHT_DEFAULT", 5),
		WeightCritical: getEnvInt("QUEUE_WEIGHT_CRITICAL", 10),

		MaxBodyBytes:      int64(getEnvInt("HTTP_MAX_BODY_BYTES", 1_048_576)), // 1 MiB
		ReadHeaderTimeout: time.Duration(getEnvInt("HTTP_READ_HEADER_TIMEOUT_MS", 5000)) * time.Millisecond,

		EnqueueAuthToken: getEnvAny([]string{"ENQUEUE_AUTH_TOKEN", "QUEUE_API_AUTH_TOKEN"}, ""),

		RedisDialTimeout:  time.Duration(getEnvInt("REDIS_DIAL_TIMEOUT_MS", 2000)) * time.Millisecond,
		RedisReadTimeout:  time.Duration(getEnvInt("REDIS_READ_TIMEOUT_MS", 1000)) * time.Millisecond,
		RedisWriteTimeout: time.Duration(getEnvInt("REDIS_WRITE_TIMEOUT_MS", 1000)) * time.Millisecond,
		RedisPingTimeout:  time.Duration(getEnvInt("REDIS_PING_TIMEOUT_MS", 1500)) * time.Millisecond,
	}

	cfg.QueueAllowList = getEnvList("QUEUE_ALLOWLIST", []string{cfg.QueueDefault, cfg.QueueCritical})
	return cfg
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

func getEnvAny(keys []string, def string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return def
}

func getEnvList(key string, def []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return def
	}
	return out
}

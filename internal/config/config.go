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

	// task allowlist / modules
	AllowedTaskTypes []string
	Modules          []string

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

// LoadWithPrefix reads configuration using the provided service prefix (e.g., "QUEUE_API", "WORKER", "SCHEDULER").
// Prefixed environment variables override unprefixed ones.
func LoadWithPrefix(prefix string) Config {
	prefix = normalizePrefix(prefix)

	cfg := Config{
		HTTPAddr:  getEnvAny(prefix, []string{"HTTP_ADDR"}, ":8081"),
		RedisAddr: getEnvAny(prefix, []string{"REDIS_ADDR"}, "redis:6379"),
		RedisPass: getEnvAny(prefix, []string{"REDIS_PASSWORD"}, ""),
		RedisDB:   getEnvIntAny(prefix, []string{"REDIS_DB"}, 0),

		Concurrency: getEnvIntAny(prefix, []string{"ASYNQ_CONCURRENCY"}, 20),

		QueueDefault:  getEnvAny(prefix, []string{"QUEUE_DEFAULT"}, "default"),
		QueueCritical: getEnvAny(prefix, []string{"QUEUE_CRITICAL"}, "critical"),

		WeightDefault:  getEnvIntAny(prefix, []string{"QUEUE_WEIGHT_DEFAULT"}, 5),
		WeightCritical: getEnvIntAny(prefix, []string{"QUEUE_WEIGHT_CRITICAL"}, 10),

		MaxBodyBytes:      int64(getEnvIntAny(prefix, []string{"HTTP_MAX_BODY_BYTES"}, 1_048_576)), // 1 MiB
		ReadHeaderTimeout: time.Duration(getEnvIntAny(prefix, []string{"HTTP_READ_HEADER_TIMEOUT_MS"}, 5000)) * time.Millisecond,

		EnqueueAuthToken: getEnvAny(prefix, []string{"ENQUEUE_AUTH_TOKEN", "QUEUE_API_AUTH_TOKEN"}, ""),

		RedisDialTimeout:  time.Duration(getEnvIntAny(prefix, []string{"REDIS_DIAL_TIMEOUT_MS"}, 2000)) * time.Millisecond,
		RedisReadTimeout:  time.Duration(getEnvIntAny(prefix, []string{"REDIS_READ_TIMEOUT_MS"}, 1000)) * time.Millisecond,
		RedisWriteTimeout: time.Duration(getEnvIntAny(prefix, []string{"REDIS_WRITE_TIMEOUT_MS"}, 1000)) * time.Millisecond,
		RedisPingTimeout:  time.Duration(getEnvIntAny(prefix, []string{"REDIS_PING_TIMEOUT_MS"}, 1500)) * time.Millisecond,
	}

	queuesDefault := []string{cfg.QueueDefault, cfg.QueueCritical}
	cfg.QueueAllowList = getEnvListAny(prefix, []string{"EDGE_ALLOWED_QUEUES", "QUEUE_ALLOWLIST"}, queuesDefault)
	cfg.AllowedTaskTypes = getEnvListAny(prefix, []string{"EDGE_ALLOWED_TASK_TYPES"}, nil)
	cfg.Modules = getEnvListAny(prefix, []string{"EDGE_MODULES"}, nil)

	return cfg
}

// Load preserves the previous signature and uses unprefixed environment variables.
func Load() Config {
	return LoadWithPrefix("")
}

func normalizePrefix(prefix string) string {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return ""
	}
	p = strings.TrimSuffix(p, "_")
	p = strings.ToUpper(p)
	return p + "_"
}

func getEnvAny(prefix string, keys []string, def string) string {
	for _, key := range keys {
		if v, ok := os.LookupEnv(prefix + key); ok {
			return v
		}
	}
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
	}
	return def
}

func getEnvIntAny(prefix string, keys []string, def int) int {
	for _, key := range keys {
		if raw, ok := os.LookupEnv(prefix + key); ok {
			return parseInt(raw, def)
		}
	}
	for _, key := range keys {
		if raw, ok := os.LookupEnv(key); ok {
			return parseInt(raw, def)
		}
	}
	return def
}

func getEnvBoolAny(prefix string, keys []string, def bool) bool {
	for _, key := range keys {
		if raw, ok := os.LookupEnv(prefix + key); ok {
			return parseBool(raw, def)
		}
	}
	for _, key := range keys {
		if raw, ok := os.LookupEnv(key); ok {
			return parseBool(raw, def)
		}
	}
	return def
}

func getEnvListAny(prefix string, keys []string, def []string) []string {
	for _, key := range keys {
		if raw, ok := os.LookupEnv(prefix + key); ok {
			return parseList(raw, def)
		}
	}
	for _, key := range keys {
		if raw, ok := os.LookupEnv(key); ok {
			return parseList(raw, def)
		}
	}
	return def
}

func parseInt(raw string, def int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func parseBool(raw string, def bool) bool {
	if raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return b
}

func parseList(raw string, def []string) []string {
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

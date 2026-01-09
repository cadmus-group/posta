package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/queue"
)

type Server struct {
	log           *slog.Logger
	cfg           config.Config
	qc            *queue.Client
	readyClient   *redis.Client
	allowedQueues map[string]struct{}
	auth          *Authenticator
	handler       http.Handler
}

func New(log *slog.Logger, cfg config.Config, qc *queue.Client) *Server {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	readyClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPass,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisDialTimeout,
		ReadTimeout:  cfg.RedisReadTimeout,
		WriteTimeout: cfg.RedisWriteTimeout,
	})

	s := &Server{
		log:           log,
		cfg:           cfg,
		qc:            qc,
		readyClient:   readyClient,
		allowedQueues: makeAllowedQueueSet(cfg.QueueAllowList),
		auth:          newAuthenticator(log, cfg.EnqueueAuthToken),
	}

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/ready", s.handleReady)

	enqueueHandler := http.HandlerFunc(s.handleEnqueue)
	if s.auth != nil {
		r.Post("/v1/enqueue", func(w http.ResponseWriter, r *http.Request) {
			s.auth.Middleware(enqueueHandler).ServeHTTP(w, r)
		})
	} else {
		r.Post("/v1/enqueue", enqueueHandler)
	}

	s.handler = r
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

type enqueueReq struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Queue   string          `json:"queue"`
}

func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request) {
	reqID := chimw.GetReqID(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	defer r.Body.Close()

	var req enqueueReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Extra garbage after JSON object?
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Queue = strings.TrimSpace(req.Queue)

	if req.Type == "" {
		http.Error(w, "missing type", http.StatusBadRequest)
		return
	}
	if req.Queue == "" {
		req.Queue = s.cfg.QueueDefault
	}

	if _, ok := s.allowedQueues[req.Queue]; !ok {
		http.Error(w, "unknown queue", http.StatusBadRequest)
		s.log.Warn("enqueue rejected: queue not allowed", "request_id", reqID, "queue", req.Queue)
		return
	}

	if !queue.IsAllowedType(req.Type) {
		http.Error(w, "unknown task type", http.StatusBadRequest)
		s.log.Warn("enqueue rejected: task type not allowed", "request_id", reqID, "type", req.Type, "queue", req.Queue)
		return
	}

	if err := queue.ValidatePayload(req.Type, req.Payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		s.log.Warn("enqueue rejected: invalid payload", "request_id", reqID, "type", req.Type, "queue", req.Queue, "err", err)
		return
	}

	task := asynq.NewTask(req.Type, req.Payload)

	info, err := s.qc.Enqueue(task, asynq.Queue(req.Queue))
	if err != nil {
		s.log.Error("enqueue failed", "request_id", reqID, "type", req.Type, "queue", req.Queue, "err", err)
		status := http.StatusInternalServerError
		if errors.Is(err, asynq.ErrQueueNotFound) {
			status = http.StatusBadRequest
		}
		http.Error(w, "enqueue failed", status)
		return
	}

	s.log.Info("enqueue ok", "request_id", reqID, "type", req.Type, "queue", req.Queue, "task_id", info.ID)

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":    info.ID,
		"queue": info.Queue,
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RedisPingTimeout)
	defer cancel()

	if err := s.readyClient.Ping(ctx).Err(); err != nil {
		s.log.Error("ready check failed", "err", err)
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func makeAllowedQueueSet(queues []string) map[string]struct{} {
	out := make(map[string]struct{}, len(queues))
	for _, q := range queues {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		out[q] = struct{}{}
	}
	return out
}

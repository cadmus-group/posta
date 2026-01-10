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

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type Server struct {
	log           *slog.Logger
	cfg           config.Config
	qc            *queue.Client
	registry      *queue.Registry
	readyClient   *redis.Client
	allowedQueues map[string]struct{}
	auth          *Authenticator
	handler       http.Handler
}

func New(log *slog.Logger, cfg config.Config, qc *queue.Client, reg *queue.Registry) *Server {
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

	if reg == nil {
		reg = queue.NewRegistry()
	}

	s := &Server{
		log:           log,
		cfg:           cfg,
		qc:            qc,
		registry:      reg,
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

	r.Post("/v1/enqueue", s.handleEnqueue)

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

	if s.auth != nil && !s.auth.isAuthorized(r) {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	defer r.Body.Close()

	var req enqueueReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}

	// Extra garbage after JSON object?
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Queue = strings.TrimSpace(req.Queue)

	if req.Type == "" {
		s.writeError(w, r, http.StatusBadRequest, "invalid_request", "missing type")
		return
	}
	if req.Queue == "" {
		req.Queue = s.cfg.QueueDefault
	}

	if _, ok := s.allowedQueues[req.Queue]; !ok {
		s.log.Warn("enqueue rejected: queue not allowed", "request_id", reqID, "queue", req.Queue)
		s.writeError(w, r, http.StatusBadRequest, "queue_not_allowed", "queue not allowed")
		return
	}

	if err := s.registry.Validate(req.Type, req.Payload); err != nil {
		code := "invalid_payload"
		message := "invalid payload"
		if errors.Is(err, queue.ErrUnknownTaskType) {
			code = "unknown_task"
			message = "unknown task type"
		}
		s.log.Warn("enqueue rejected: invalid payload", "request_id", reqID, "type", req.Type, "queue", req.Queue, "err", err)
		s.writeError(w, r, http.StatusBadRequest, code, message)
		return
	}

	task := asynq.NewTask(req.Type, req.Payload)

	info, err := s.qc.Enqueue(task, asynq.Queue(req.Queue))
	if err != nil {
		s.log.Error("enqueue failed", "request_id", reqID, "type", req.Type, "queue", req.Queue, "err", err)
		status := http.StatusInternalServerError
		code := "enqueue_failed"
		message := "enqueue failed"
		if errors.Is(err, asynq.ErrQueueNotFound) {
			status = http.StatusBadRequest
			code = "queue_not_found"
			message = "queue not found"
		}
		s.writeError(w, r, status, code, message)
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

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{
		Code:      code,
		Message:   msg,
		RequestID: chimw.GetReqID(r.Context()),
	})
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

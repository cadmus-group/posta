package httpapi

import (
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

	"github.com/cadmus-group/posta/internal/config"
	"github.com/cadmus-group/posta/internal/queue"
)

type Server struct {
	log      *slog.Logger
	cfg      config.Config
	qc       *queue.Client
	handler  http.Handler
}

func New(log *slog.Logger, cfg config.Config, qc *queue.Client) *Server {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	s := &Server{log: log, cfg: cfg, qc: qc}

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

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

	task := asynq.NewTask(req.Type, req.Payload)

	info, err := s.qc.Enqueue(task, asynq.Queue(req.Queue))
	if err != nil {
		s.log.Error("enqueue failed", "err", err)
		status := http.StatusInternalServerError
		if errors.Is(err, asynq.ErrQueueNotFound) {
			status = http.StatusBadRequest
		}
		http.Error(w, "enqueue failed", status)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":    info.ID,
		"queue": info.Queue,
	})
}

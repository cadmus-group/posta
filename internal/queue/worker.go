package queue

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/hibiken/asynq"
)

type Worker struct {
	log *slog.Logger
}

func NewWorker(log *slog.Logger) *Worker {
	return &Worker{log: log}
}

func (w *Worker) Mux() *asynq.ServeMux {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TypeExamplePing, w.handlePing)
	mux.HandleFunc(TypeExampleEmail, w.handleSendEmail)

	return mux
}

func (w *Worker) handlePing(ctx context.Context, t *asynq.Task) error {
	w.log.Info("handled ping", "payload", string(t.Payload()))
	return nil
}

func (w *Worker) handleSendEmail(ctx context.Context, t *asynq.Task) error {
	var p SendEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Replace with real email provider integration.
	w.log.Info("sending email", "to", p.To, "subject", p.Subject)
	return nil
}

package demo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/queue"
)

const (
	TypeExampleEmail = "example:send_email"
	TypeExamplePing  = "example:ping"
)

type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func Register(log *slog.Logger, reg *queue.Registry) error {
	reg.MustRegister(queue.TaskSpec{
		Type:     TypeExamplePing,
		Validate: validatePingPayload,
		Handler:  handlePing(log),
	})

	reg.MustRegister(queue.TaskSpec{
		Type:     TypeExampleEmail,
		Validate: validateSendEmailPayload,
		Handler:  handleSendEmail(log),
	})

	return nil
}

func RegisterSchedules(log *slog.Logger, s *asynq.Scheduler, defaultQueue string) error {
	task := asynq.NewTask(TypeExamplePing, []byte(`{"hello":"world"}`))
	if _, err := s.Register("@every 1m", task, asynq.Queue(defaultQueue)); err != nil {
		return err
	}
	log.Info("demo schedule registered", "task_type", TypeExamplePing)
	return nil
}

func handlePing(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("handled ping", "payload", string(t.Payload()))
		return nil
	}
}

func handleSendEmail(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p SendEmailPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		log.Info("sending email", "to", p.To, "subject", p.Subject)
		return nil
	}
}

func validatePingPayload(payload json.RawMessage) error {
	if len(payload) == 0 {
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return errors.New("invalid payload")
	}
	return nil
}

func validateSendEmailPayload(payload json.RawMessage) error {
	if len(payload) == 0 {
		return errors.New("payload required")
	}

	var p SendEmailPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return errors.New("invalid payload")
	}

	p.To = strings.TrimSpace(p.To)
	p.Subject = strings.TrimSpace(p.Subject)
	p.Body = strings.TrimSpace(p.Body)

	switch {
	case p.To == "" || p.Subject == "" || p.Body == "":
		return errors.New("missing required fields")
	case len(p.To) > 320:
		return errors.New("to too long")
	case len(p.Subject) > 256:
		return errors.New("subject too long")
	case len(p.Body) > 8000:
		return errors.New("body too long")
	}

	return nil
}

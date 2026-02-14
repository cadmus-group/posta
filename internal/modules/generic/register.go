package generic

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/queue"
)

// Task type for generic/passthrough payloads (e.g. testing or edge use).
const TypeGeneric = "generic"

func Register(log *slog.Logger, reg *queue.Registry) error {
	reg.MustRegister(queue.TaskSpec{
		Type:     TypeGeneric,
		Validate: func(_ []byte) error { return nil }, // accept any JSON
		Handler: func(ctx context.Context, t *asynq.Task) error {
			log.Info("generic task", "payload", string(t.Payload()))
			return nil
		},
	})
	return nil
}

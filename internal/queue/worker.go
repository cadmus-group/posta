package queue

import "github.com/hibiken/asynq"

// NewServeMux constructs an asynq ServeMux from the registry, optionally filtered to allowed task types.
func NewServeMux(reg *Registry, allowed []string) (*asynq.ServeMux, error) {
	return reg.ServeMux(allowed)
}

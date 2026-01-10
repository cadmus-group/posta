package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hibiken/asynq"
)

type Validator func(json.RawMessage) error

type TaskSpec struct {
	Type     string
	Validate Validator
	Handler  asynq.HandlerFunc
}

type Registry struct {
	validators map[string]Validator
	handlers   map[string]asynq.HandlerFunc
}

var ErrUnknownTaskType = errors.New("unknown task type")

func NewRegistry() *Registry {
	return &Registry{
		validators: make(map[string]Validator),
		handlers:   make(map[string]asynq.HandlerFunc),
	}
}

func (r *Registry) Register(spec TaskSpec) error {
	if spec.Type = strings.TrimSpace(spec.Type); spec.Type == "" {
		return errors.New("task type is required")
	}
	if _, exists := r.validators[spec.Type]; exists {
		return fmt.Errorf("task type already registered: %s", spec.Type)
	}
	r.validators[spec.Type] = spec.Validate
	if spec.Handler != nil {
		r.handlers[spec.Type] = spec.Handler
	}
	return nil
}

func (r *Registry) MustRegister(spec TaskSpec) {
	if err := r.Register(spec); err != nil {
		panic(err)
	}
}

func (r *Registry) AllowedTypes() []string {
	out := make([]string, 0, len(r.validators))
	for t := range r.validators {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) Validate(taskType string, payload json.RawMessage) error {
	v, ok := r.validators[taskType]
	if !ok {
		return ErrUnknownTaskType
	}
	if v == nil {
		return nil
	}
	return v(payload)
}

func (r *Registry) HasHandler(taskType string) bool {
	_, ok := r.handlers[taskType]
	return ok
}

func (r *Registry) Handler(taskType string) (asynq.HandlerFunc, bool) {
	h, ok := r.handlers[taskType]
	return h, ok
}

func (r *Registry) Filter(allowed []string) (*Registry, error) {
	if len(allowed) == 0 {
		return r, nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, t := range allowed {
		t = strings.TrimSpace(t)
		if t != "" {
			allowedSet[t] = struct{}{}
		}
	}

	next := NewRegistry()
	for t, v := range r.validators {
		if _, ok := allowedSet[t]; ok {
			next.validators[t] = v
			if h, ok := r.handlers[t]; ok {
				next.handlers[t] = h
			}
			delete(allowedSet, t)
		}
	}

	if len(allowedSet) > 0 {
		missing := make([]string, 0, len(allowedSet))
		for t := range allowedSet {
			missing = append(missing, t)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("allowed task types not registered: %s", strings.Join(missing, ", "))
	}

	return next, nil
}

func (r *Registry) ServeMux(allowed []string) (*asynq.ServeMux, error) {
	reg, err := r.Filter(allowed)
	if err != nil {
		return nil, err
	}

	mux := asynq.NewServeMux()
	for t, h := range reg.handlers {
		mux.HandleFunc(t, h)
	}
	return mux, nil
}

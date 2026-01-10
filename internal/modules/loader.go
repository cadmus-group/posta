package modules

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/cadmus-group/posta/internal/queue"
	demomod "github.com/cadmus-group/posta/modules/demo"
)

type moduleSpec struct {
	registerTasks     func(log *slog.Logger, reg *queue.Registry) error
	registerSchedules func(log *slog.Logger, s *asynq.Scheduler, defaultQueue string) error
}

var moduleCatalog = map[string]moduleSpec{
	"demo": {
		registerTasks:     demomod.Register,
		registerSchedules: demomod.RegisterSchedules,
	},
}

// BuildRegistry loads modules and optionally filters to the allowed task types list.
func BuildRegistry(modules []string, allowedTaskTypes []string, log *slog.Logger) (*queue.Registry, error) {
	reg := queue.NewRegistry()
	if err := LoadModules(modules, log, reg); err != nil {
		return nil, err
	}
	return reg.Filter(allowedTaskTypes)
}

func LoadModules(modules []string, log *slog.Logger, reg *queue.Registry) error {
	names := normalizeModules(modules)
	for _, name := range names {
		spec, ok := moduleCatalog[name]
		if !ok {
			return fmt.Errorf("unknown module: %s", name)
		}
		if spec.registerTasks != nil {
			if err := spec.registerTasks(log, reg); err != nil {
				return fmt.Errorf("module %s registration failed: %w", name, err)
			}
		}
	}
	return nil
}

func RegisterModuleSchedules(modules []string, log *slog.Logger, s *asynq.Scheduler, defaultQueue string) error {
	names := normalizeModules(modules)
	for _, name := range names {
		spec, ok := moduleCatalog[name]
		if !ok {
			return fmt.Errorf("unknown module: %s", name)
		}
		if spec.registerSchedules == nil {
			continue
		}
		if err := spec.registerSchedules(log, s, defaultQueue); err != nil {
			return fmt.Errorf("module %s schedule registration failed: %w", name, err)
		}
	}
	return nil
}

func normalizeModules(modules []string) []string {
	seen := make(map[string]struct{}, len(modules))
	out := make([]string, 0, len(modules))
	for _, m := range modules {
		m = strings.ToLower(strings.TrimSpace(m))
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

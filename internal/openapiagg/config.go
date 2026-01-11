package openapiagg

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultRefreshSeconds = 30
	defaultConfigPath     = "config/openapi-sources.yaml"
)

// DefaultConfigPath returns the default config location.
func DefaultConfigPath() string {
	return defaultConfigPath
}

// Source describes one microservice that exposes an OpenAPI document.
type Source struct {
	Name            string `yaml:"name" json:"name"`
	OpenAPIURL      string `yaml:"openapiUrl" json:"openapiUrl"`
	PathPrefix      string `yaml:"pathPrefix" json:"pathPrefix"`
	ComponentPrefix string `yaml:"componentPrefix" json:"componentPrefix"`
	OperationPrefix string `yaml:"operationPrefix" json:"operationPrefix"`
}

// Config is the root config file consumed by the aggregator.
type Config struct {
	RefreshSeconds int      `yaml:"refreshSeconds" json:"refreshSeconds"`
	Sources        []Source `yaml:"sources" json:"sources"`
}

// LoadConfig reads and validates the config from disk.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.RefreshSeconds <= 0 {
		cfg.RefreshSeconds = defaultRefreshSeconds
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate ensures the config is usable.
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return errors.New("no sources configured")
	}
	for i := range c.Sources {
		if err := c.Sources[i].validate(); err != nil {
			return fmt.Errorf("source %q: %w", c.Sources[i].Name, err)
		}
	}
	return nil
}

func (s *Source) validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.OpenAPIURL == "" {
		return errors.New("openapiUrl is required")
	}
	if s.PathPrefix == "" {
		return errors.New("pathPrefix is required to avoid collisions")
	}
	return nil
}

// ComponentPrefixSafe returns the prefix to apply to component names.
func (s *Source) ComponentPrefixSafe() string {
	if s.ComponentPrefix != "" {
		return s.ComponentPrefix
	}
	return toTitleNoSpace(s.Name)
}

// OperationPrefixSafe returns the prefix to apply to operationIds.
func (s *Source) OperationPrefixSafe() string {
	if s.OperationPrefix != "" {
		return s.OperationPrefix
	}
	return s.ComponentPrefixSafe()
}

func toTitleNoSpace(in string) string {
	if in == "" {
		return ""
	}
	in = strings.TrimSpace(in)
	if len(in) == 1 {
		return strings.ToUpper(in)
	}
	return strings.ToUpper(in[:1]) + in[1:]
}

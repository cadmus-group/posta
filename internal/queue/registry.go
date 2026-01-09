package queue

import (
	"encoding/json"
	"errors"
	"strings"
)

type validator func(json.RawMessage) error

var registry = map[string]validator{
	TypeExamplePing:  validatePingPayload,
	TypeExampleEmail: validateSendEmailPayload,
}

func IsAllowedType(taskType string) bool {
	_, ok := registry[taskType]
	return ok
}

func ValidatePayload(taskType string, payload json.RawMessage) error {
	v, ok := registry[taskType]
	if !ok {
		return errors.New("unknown task type")
	}

	if v == nil {
		return nil
	}

	return v(payload)
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

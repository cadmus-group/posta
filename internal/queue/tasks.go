package queue

import (
	"encoding/json"
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

func MustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

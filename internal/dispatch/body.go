package dispatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"hookhub/internal/store"
)

type bodyData struct {
	Subscription *store.Subscription
	Event        string
	Payload      string
	SentAt       string
}

type envelope struct {
	Type    string          `json:"type"`
	SentAt  string          `json:"sent_at"`
	Payload json.RawMessage `json:"payload"`
}

func EnvelopeBody(eventType string, payload []byte) ([]byte, error) {
	raw := payload
	if len(raw) == 0 {
		raw = []byte("null")
	}
	return json.Marshal(envelope{Type: eventType, SentAt: time.Now().UTC().Format(time.RFC3339), Payload: raw})
}

func BuildBody(tmplSrc string, sub *store.Subscription, eventType string, payload []byte) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	t, err := template.New("body").Parse(tmplSrc)
	if err != nil {
		return nil, fmt.Errorf("parse body template: %w", err)
	}
	var buf bytes.Buffer
	data := bodyData{Subscription: sub, Event: eventType, Payload: string(payload), SentAt: now}
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render body template: %w", err)
	}
	return buf.Bytes(), nil
}

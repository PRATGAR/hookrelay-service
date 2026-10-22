package api

import (
	"encoding/json"
	"net/http"

	"hookhub/internal/guard"
)

const maxPayloadBytes = 64 << 10

func (h *Handlers) PublishEvent(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	var req struct {
		Type    guard.EventType `json:"type"`
		Payload json.RawMessage `json:"payload"`
		AckURL  string          `json:"ack_url"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid event")
		return
	}
	if len(req.Payload) == 0 || len(req.Payload) > maxPayloadBytes {
		writeError(w, http.StatusUnprocessableEntity, "payload missing or too large")
		return
	}
	results, err := h.fan.Publish(r.Context(), tenant.ID, string(req.Type), req.Payload, req.AckURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "delivery failed")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"event": req.Type, "deliveries": results})
}

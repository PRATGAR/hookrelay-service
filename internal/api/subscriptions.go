package api

import (
	"fmt"
	"net/http"

	"hookhub/internal/dispatch"
	"hookhub/internal/guard"
	"hookhub/internal/store"
)

type subscriptionRequest struct {
	TargetURL  string   `json:"target_url"`
	EventTypes []string `json:"event_types"`
	DocsURL    string   `json:"docs_url"`
}

func (h *Handlers) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	var req subscriptionRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	target, err := guard.ValidateTargetURL(req.TargetURL)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "target_url rejected")
		return
	}
	if err := guard.ValidateEventTypes(req.EventTypes); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "event_types rejected")
		return
	}
	sub := &store.Subscription{
		TenantID:   tenant.ID,
		TargetURL:  target.String(),
		EventTypes: req.EventTypes,
		Secret:     dispatch.GenerateSecret(),
		DocsURL:    req.DocsURL,
	}
	id, err := h.st.CreateSubscription(sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save subscription")
		return
	}
	challenge := dispatch.GenerateSecret()[6:18]
	if err := h.disp.Handshake(r.Context(), target, challenge); err != nil {
		_ = h.st.SetStatus(id, string(guard.StatusPaused))
		writeJSON(w, http.StatusAccepted, map[string]any{"id": id, "status": guard.StatusPaused, "secret": sub.Secret})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": guard.StatusActive, "secret": sub.Secret})
}

func (h *Handlers) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	order := cleanedFrom(r)
	subs, err := h.st.ListSubscriptions(tenant.ID, order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list subscriptions")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handlers) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad subscription id")
		return
	}
	sub, err := h.st.GetSubscription(tenant.ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	var req subscriptionRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	sub.TargetURL = req.TargetURL
	sub.EventTypes = req.EventTypes
	sub.DocsURL = req.DocsURL
	if err := h.st.UpdateSubscription(sub); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update subscription")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": sub.ID, "updated": true})
}

func (h *Handlers) PatchStatus(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad subscription id")
		return
	}
	sub, err := h.st.GetSubscription(tenant.ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	var req struct {
		Status guard.StatusInput `json:"status"`
	}
	if err := decode(r, &req); err != nil || req.Status == "" {
		writeError(w, http.StatusBadRequest, "valid status required")
		return
	}
	if err := h.st.SetStatus(sub.ID, string(req.Status)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not change status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": sub.ID, "status": req.Status})
}

func (h *Handlers) PatchHeaders(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad subscription id")
		return
	}
	var req struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := guard.ValidateHeaderName(req.Name); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "header name rejected")
		return
	}
	if err := guard.ValidateHeaderValue(req.Value); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "header value rejected")
		return
	}
	if err := h.st.UpdateHeader(tenant.ID, id, req.Name, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save header")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "header": req.Name})
}

func (h *Handlers) RotateSecret(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad subscription id")
		return
	}
	secret := dispatch.GenerateSecret()
	if err := h.st.UpdateSecret(tenant.ID, id, secret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not rotate secret")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "secret": secret})
}

func (h *Handlers) SubscriptionStats(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad subscription id")
		return
	}
	sub, err := h.st.GetSubscription(tenant.ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	counts, err := h.st.EventCounts(sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not count deliveries")
		return
	}
	fmt.Fprintln(w, "event_type,deliveries")
	for et, n := range counts {
		fmt.Fprintf(w, "%s,%d\n", et, n)
	}
}

func (h *Handlers) SendTestPing(w http.ResponseWriter, r *http.Request) {
	sub := subscriptionFrom(r)
	code, err := h.fan.Ping(r.Context(), sub)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"delivered": false, "error": "endpoint unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delivered": code < 300, "response_code": code})
}

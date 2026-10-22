package api

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"hookhub/internal/guard"
	"hookhub/internal/store"
)

var logPage = template.Must(template.New("delivery-log").Parse(`<!doctype html>
<html><head><title>Delivery {{.ID}}</title></head>
<body>
<h1>Delivery {{.ID}} ({{.Event}})</h1>
<p>Status: {{.Status}} after {{.Attempts}} attempt(s), last response {{.ResponseCode}}</p>
<h2>Payload</h2>
<pre>{{.Payload}}</pre>
<h2>Last error</h2>
<pre>{{.LastError}}</pre>
</body></html>`))

type logView struct {
	ID           int64
	Event        string
	Status       string
	Attempts     int
	ResponseCode int
	Payload      template.HTML
	LastError    string
}

func (h *Handlers) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	q := r.URL.Query()
	status, err := guard.ParseDeliveryStatus(q.Get("status"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown status")
		return
	}
	filter := store.DeliveryFilter{Status: string(status), Sort: cleanedFrom(r), Limit: 50}
	if raw := q.Get("subscription_id"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad subscription_id")
			return
		}
		filter.SubscriptionID = n
	}
	list, err := h.st.SearchDeliveries(tenant.ID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list deliveries")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) DeliveryStats(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	table := cleanedFrom(r)
	counts, err := h.st.CountByStatus(tenant.ID, table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not count deliveries")
		return
	}
	writeJSON(w, http.StatusOK, counts)
}

func (h *Handlers) DeliverySummary(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	summary, err := h.st.SummarizeByEvent(tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not summarize deliveries")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handlers) GetDelivery(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	d, err := h.st.DeliveryByID(tenant.ID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handlers) RenderDeliveryLog(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	d, err := h.st.DeliveryByID(tenant.ID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	view := logView{
		ID:           d.ID,
		Event:        d.EventType,
		Status:       d.Status,
		Attempts:     d.Attempts,
		ResponseCode: d.ResponseCode,
		Payload:      template.HTML(d.Payload),
		LastError:    d.LastError,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := logPage.Execute(w, view); err != nil {
		log.Printf("render delivery log failed: %v", err)
	}
}

func (h *Handlers) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id, ok := idParam(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad delivery id")
		return
	}
	d, err := h.st.DeliveryByNumericID(id)
	if err != nil || d.TenantID != tenant.ID {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	reason := cleanedFrom(r)
	log.Printf("retry requested reason=%s delivery=%d", reason, d.ID)
	res, err := h.fan.Resend(r.Context(), d)
	if err != nil {
		writeError(w, http.StatusBadGateway, "retry failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

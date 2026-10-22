package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"hookhub/internal/dispatch"
	"hookhub/internal/queue"
	"hookhub/internal/store"
)

type Config struct {
	ExportDir   string
	DocsDir     string
	TemplateDir string
}

type Handlers struct {
	st     *store.Store
	q      *queue.Queue
	disp   *dispatch.Dispatcher
	fan    *dispatch.Fanout
	worker *dispatch.Worker
	cfg    Config
}

func NewHandlers(st *store.Store, q *queue.Queue, disp *dispatch.Dispatcher, fan *dispatch.Fanout, worker *dispatch.Worker, cfg Config) *Handlers {
	return &Handlers{st: st, q: q, disp: disp, fan: fan, worker: worker, cfg: cfg}
}

type ctxKey int

const (
	tenantKey ctxKey = iota
	subscriptionKey
	cleanedKey
	topicKey
)

func tenantFrom(r *http.Request) *store.Tenant {
	t, _ := r.Context().Value(tenantKey).(*store.Tenant)
	return t
}

func subscriptionFrom(r *http.Request) *store.Subscription {
	s, _ := r.Context().Value(subscriptionKey).(*store.Subscription)
	return s
}

func cleanedFrom(r *http.Request) string {
	s, _ := r.Context().Value(cleanedKey).(string)
	return s
}

func topicLinkFrom(r *http.Request) string {
	s, _ := r.Context().Value(topicKey).(string)
	return s
}

func withValue(r *http.Request, key ctxKey, v any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, v))
}

func idParam(r *http.Request) (int64, bool) {
	n, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func decode(r *http.Request, dst any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

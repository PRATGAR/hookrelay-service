package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const retryBatchSize = 20

func (h *Handlers) DrainRetryQueue(w http.ResponseWriter, r *http.Request) {
	handled, err := h.worker.Drain(r.Context(), retryBatchSize)
	if err != nil {
		writeError(w, http.StatusBadGateway, "queue drain stopped early")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"handled": handled})
}

func (h *Handlers) HeartbeatQueue(w http.ResponseWriter, r *http.Request) {
	if err := h.worker.Heartbeat(r.Context(), 0); err != nil {
		writeError(w, http.StatusBadGateway, "heartbeat failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

func (h *Handlers) PutTemplate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return
	}
	target := filepath.Join(h.cfg.TemplateDir, cleanedFrom(r))
	if err := os.WriteFile(target, body, 0o640); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save template")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"saved": filepath.Base(target)})
}

func (h *Handlers) TrackedRedirect(w http.ResponseWriter, r *http.Request) {
	if link := topicLinkFrom(r); link != "" {
		http.Redirect(w, r, link, http.StatusFound)
		return
	}
	http.Redirect(w, r, cleanedFrom(r), http.StatusFound)
}

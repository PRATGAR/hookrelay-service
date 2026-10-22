package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"hookhub/internal/guard"
	"hookhub/internal/store"
)

func (h *Handlers) CreateExport(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	var req struct {
		Name   guard.ExportLabel `json:"name"`
		Status string            `json:"status"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "valid name required")
		return
	}
	status, err := guard.ParseDeliveryStatus(req.Status)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown status")
		return
	}
	rows, err := h.st.SearchDeliveries(tenant.ID, store.DeliveryFilter{Status: string(status), Sort: "created_at DESC", Limit: 1000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read deliveries")
		return
	}
	f, err := os.Create(filepath.Join(h.cfg.ExportDir, string(req.Name)+".csv"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create export")
		return
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	_ = cw.Write([]string{"id", "subscription_id", "event_type", "status", "attempts", "response_code"})
	for _, d := range rows {
		_ = cw.Write([]string{
			fmt.Sprint(d.ID), fmt.Sprint(d.SubscriptionID), d.EventType, d.Status,
			fmt.Sprint(d.Attempts), fmt.Sprint(d.ResponseCode),
		})
	}
	cw.Flush()
	writeJSON(w, http.StatusCreated, map[string]any{"file": string(req.Name) + ".csv", "rows": len(rows)})
}

func (h *Handlers) GetExport(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(filepath.Join(h.cfg.ExportDir, chi.URLParam(r, "file")))
	if err != nil {
		writeError(w, http.StatusNotFound, "export not found")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	_, _ = io.Copy(w, f)
}

func (h *Handlers) GetDoc(w http.ResponseWriter, r *http.Request) {
	raw, err := os.ReadFile(filepath.Join(h.cfg.DocsDir, cleanedFrom(r)))
	if err != nil {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = io.Copy(w, bytes.NewReader(raw))
}

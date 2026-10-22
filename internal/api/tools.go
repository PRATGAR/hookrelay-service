package api

import (
	"bytes"
	"io"
	"net/http"
	"os/exec"

	"hookhub/internal/dispatch"
	"hookhub/internal/guard"
)

func (h *Handlers) FetchSample(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL guard.FetchURL `json:"url"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "url rejected")
		return
	}
	sample, err := dispatch.FetchSample(string(req.URL))
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not fetch sample")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, bytes.NewReader(sample))
}

func lookupHost(host string) (string, error) {
	out, err := exec.Command("dig", host, "+short").Output()
	return string(out), err
}

func (h *Handlers) DNSVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host guard.Hostname `json:"host"`
	}
	if err := decode(r, &req); err != nil || req.Host == "" {
		writeError(w, http.StatusBadRequest, "valid host required")
		return
	}
	answer, err := lookupHost(string(req.Host))
	if err != nil {
		writeError(w, http.StatusBadGateway, "lookup failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"host": string(req.Host), "answer": answer})
}

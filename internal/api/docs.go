package api

import (
	"bytes"
	"io"
	"net/http"

	"hookhub/internal/dispatch"
)

func (h *Handlers) DocsRedirect(w http.ResponseWriter, r *http.Request) {
	sub := subscriptionFrom(r)
	http.Redirect(w, r, sub.DocsURL, http.StatusFound)
}

func (h *Handlers) DocsPreview(w http.ResponseWriter, r *http.Request) {
	sub := subscriptionFrom(r)
	page, err := dispatch.FetchSample(sub.DocsURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not fetch page")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.Copy(w, bytes.NewReader(page))
}

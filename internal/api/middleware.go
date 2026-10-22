package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hookhub/internal/guard"
)

func (h *Handlers) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Api-Key")
		if key == "" {
			writeError(w, http.StatusUnauthorized, "missing api key")
			return
		}
		tenant, err := h.st.TenantByKeyHash(guard.DigestKey(key))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unknown api key")
			return
		}
		next.ServeHTTP(w, withValue(r, tenantKey, tenant))
	})
}

func (h *Handlers) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t := tenantFrom(r); t == nil || t.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handlers) EgressGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := idParam(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "bad subscription id")
			return
		}
		sub, err := h.st.GetSubscription(tenantFrom(r).ID, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		if err := guard.CheckEgress(sub.TargetURL); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "target not reachable from here")
			return
		}
		next.ServeHTTP(w, withValue(r, subscriptionKey, sub))
	})
}

func ExportFileName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !guard.ValidExportName(chi.URLParam(r, "file")) {
			writeError(w, http.StatusBadRequest, "unknown export")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CSVResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"export.csv\"")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func cleaning(key ctxKey, clean func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withValue(r, key, clean(r)))
		})
	}
}

var DocPath = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.NormalizeDocPath(chi.URLParam(r, "*"))
})

var RedirectTarget = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.SameSiteOnly(r.URL.Query().Get("to"))
})

var RetryReason = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.OneLine(r.URL.Query().Get("reason"))
})

var TemplateName = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.TemplateFileName(chi.URLParam(r, "name"))
})

var TopicLink = cleaning(topicKey, func(r *http.Request) string {
	if topic := r.URL.Query().Get("topic"); topic != "" {
		return guard.DocsLink(topic)
	}
	return ""
})

var SubscriptionSort = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.SubscriptionOrder(r.URL.Query().Get("sort"))
})

var DeliveryScope = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.DeliveryTable(r.URL.Query().Get("scope"))
})

var DeliveryOrder = cleaning(cleanedKey, func(r *http.Request) string {
	return guard.NormalizeOrder(r.URL.Query().Get("sort"))
})

func (h *Handlers) DocsGuard(accept func(string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := idParam(r)
			if !ok {
				writeError(w, http.StatusBadRequest, "bad subscription id")
				return
			}
			sub, err := h.st.GetSubscription(tenantFrom(r).ID, id)
			if err != nil || !accept(sub.DocsURL) {
				writeError(w, http.StatusNotFound, "no documentation page")
				return
			}
			next.ServeHTTP(w, withValue(r, subscriptionKey, sub))
		})
	}
}

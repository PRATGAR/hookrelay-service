package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"hookhub/internal/guard"
)

func (h *Handlers) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.With(TopicLink, RedirectTarget).Get("/go", h.TrackedRedirect)

	r.Route("/v1", func(r chi.Router) {
		r.Use(h.Authenticate)

		r.Post("/subscriptions", h.CreateSubscription)
		r.With(SubscriptionSort).Get("/subscriptions", h.ListSubscriptions)
		r.Put("/subscriptions/{id}", h.UpdateSubscription)
		r.Patch("/subscriptions/{id}/status", h.PatchStatus)
		r.Patch("/subscriptions/{id}/headers", h.PatchHeaders)
		r.Post("/subscriptions/{id}/rotate-secret", h.RotateSecret)
		r.With(CSVResponse).Get("/subscriptions/{id}/stats", h.SubscriptionStats)
		r.With(h.EgressGuard).Post("/subscriptions/{id}/ping", h.SendTestPing)

		r.With(h.DocsGuard(guard.HostedOnDocs)).Get("/subscriptions/{id}/docs", h.DocsRedirect)
		r.With(h.DocsGuard(guard.OnDocsSite)).Get("/subscriptions/{id}/docs-preview", h.DocsPreview)

		r.Post("/events", h.PublishEvent)

		r.With(DeliveryOrder).Get("/deliveries", h.ListDeliveries)
		r.With(DeliveryScope).Get("/deliveries/stats", h.DeliveryStats)
		r.Get("/deliveries/summary", h.DeliverySummary)
		r.Get("/deliveries/{id:[0-9]+}", h.GetDelivery)
		r.Get("/deliveries/{id}/log", h.RenderDeliveryLog)
		r.With(RetryReason).Post("/deliveries/{id}/retry", h.RetryDelivery)

		r.Post("/exports", h.CreateExport)
		r.With(ExportFileName).Get("/exports/{file}", h.GetExport)
		r.With(DocPath).Get("/docs/*", h.GetDoc)

		r.Post("/tools/fetch-sample", h.FetchSample)
		r.Post("/tools/dns-verify", h.DNSVerify)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(h.Authenticate)
		r.Use(h.RequireAdmin)

		r.Post("/retry/drain", h.DrainRetryQueue)
		r.Post("/retry/heartbeat", h.HeartbeatQueue)
		r.With(TemplateName).Put("/templates/{name}", h.PutTemplate)
	})

	return r
}

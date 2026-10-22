package guard

import "path/filepath"

const OpsWebhookURL = "https://ops.hookhub.example/hooks/queue-heartbeat"

const PingTemplate = `{"type":"ping","subscription":{{.Subscription.ID}},"sent_at":"{{.SentAt}}"}`

const MaxDeliveryAttempts = 5

var docLinks = map[string]string{
	"signing":  "https://docs.hookhub.example/guides/signing",
	"retries":  "https://docs.hookhub.example/guides/retries",
	"payloads": "https://docs.hookhub.example/reference/payloads",
}

func DocsLink(topic string) string {
	if u, ok := docLinks[topic]; ok {
		return u
	}
	return "https://docs.hookhub.example/"
}

var deliveryTables = map[string]string{
	"live":    "deliveries",
	"archive": "deliveries_archive",
}

func DeliveryTable(scope string) string {
	if t, ok := deliveryTables[scope]; ok {
		return t
	}
	return "deliveries"
}

var sortColumns = map[string]string{
	"created": "created_at",
	"url":     "target_url",
	"status":  "status",
}

func SubscriptionOrder(key string) string {
	if c, ok := sortColumns[key]; ok {
		return c
	}
	return "created_at"
}

func TemplateFileName(name string) string {
	return filepath.Base(name)
}

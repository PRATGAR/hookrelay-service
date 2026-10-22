package guard

import (
	"encoding/json"
	"net/url"
)

type EventType string

func (e *EventType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if !eventTypePattern.MatchString(s) {
		return ErrRejected
	}
	*e = EventType(s)
	return nil
}

type Hostname string

func (h *Hostname) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if !hostnamePattern.MatchString(s) {
		return ErrRejected
	}
	*h = Hostname(s)
	return nil
}

type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusPaused   SubscriptionStatus = "paused"
	StatusDisabled SubscriptionStatus = "disabled"
)

func (s *SubscriptionStatus) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	switch SubscriptionStatus(raw) {
	case StatusActive, StatusPaused, StatusDisabled:
		*s = SubscriptionStatus(raw)
		return nil
	}
	return ErrRejected
}

type StatusInput string

func (s *StatusInput) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := ValidateStatus(raw); err != nil {
		return err
	}
	*s = StatusInput(raw)
	return nil
}

type ExportLabel string

func (l *ExportLabel) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := ValidateExportLabel(raw); err != nil {
		return err
	}
	*l = ExportLabel(raw)
	return nil
}

type FetchURL string

func (f *FetchURL) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ErrRejected
	}
	if IsInternalHost(u.Hostname()) {
		return ErrRejected
	}
	*f = FetchURL(raw)
	return nil
}

type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryFailed    DeliveryStatus = "failed"
)

func ParseDeliveryStatus(raw string) (DeliveryStatus, error) {
	switch DeliveryStatus(raw) {
	case "", DeliveryPending, DeliveryDelivered, DeliveryFailed:
		return DeliveryStatus(raw), nil
	}
	return "", ErrRejected
}

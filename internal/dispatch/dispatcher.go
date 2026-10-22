package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"hookhub/internal/store"
)

type Dispatcher struct {
	client *http.Client
}

func New() *Dispatcher {
	return &Dispatcher{client: &http.Client{Timeout: 10 * time.Second}}
}

func (d *Dispatcher) Deliver(ctx context.Context, sub *store.Subscription, eventType string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.TargetURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["X-Hook-Event"] = []string{eventType}
	req.Header["X-Hook-Signature"] = []string{Sign(sub.Secret, body)}
	if sub.HeaderName != "" {
		req.Header.Set(sub.HeaderName, sub.HeaderValue)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, nil
}

func (d *Dispatcher) Notify(ctx context.Context, target string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("callback answered %d", resp.StatusCode)
	}
	return nil
}

func (d *Dispatcher) Handshake(ctx context.Context, target *url.URL, challenge string) error {
	q := target.Query()
	q.Set("hook_challenge", challenge)
	target.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	echoed, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return err
	}
	if string(echoed) != challenge {
		return fmt.Errorf("endpoint did not echo the challenge")
	}
	return nil
}

func FetchSample(location string) ([]byte, error) {
	resp, err := http.Get(location)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
}

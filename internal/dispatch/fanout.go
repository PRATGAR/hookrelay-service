package dispatch

import (
	"context"
	"fmt"

	"hookhub/internal/guard"
	"hookhub/internal/queue"
	"hookhub/internal/store"
)

type Result struct {
	SubscriptionID int64  `json:"subscription_id"`
	DeliveryID     int64  `json:"delivery_id"`
	Status         string `json:"status"`
	ResponseCode   int    `json:"response_code"`
}

type Fanout struct {
	st *store.Store
	d  *Dispatcher
	q  *queue.Queue
}

func NewFanout(st *store.Store, d *Dispatcher, q *queue.Queue) *Fanout {
	return &Fanout{st: st, d: d, q: q}
}

func (f *Fanout) Publish(ctx context.Context, tenantID int64, eventType string, payload []byte, ackURL string) ([]Result, error) {
	subs, err := f.st.ListActiveByEvent(tenantID, eventType)
	if err != nil {
		return nil, fmt.Errorf("find subscriptions: %w", err)
	}
	results := make([]Result, 0, len(subs))
	for i := range subs {
		res, err := f.deliverOne(ctx, &subs[i], eventType, payload, ackURL)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (f *Fanout) deliverOne(ctx context.Context, sub *store.Subscription, eventType string, payload []byte, ackURL string) (Result, error) {
	body, err := EnvelopeBody(eventType, payload)
	if err != nil {
		return Result{}, err
	}
	id, err := f.st.InsertDelivery(&store.Delivery{
		SubscriptionID: sub.ID,
		TenantID:       sub.TenantID,
		EventType:      eventType,
		Payload:        string(payload),
	})
	if err != nil {
		return Result{}, err
	}
	if ackURL != "" {
		if err := f.q.SetAckURL(ctx, id, ackURL); err != nil {
			return Result{}, err
		}
	}
	res := Result{SubscriptionID: sub.ID, DeliveryID: id, Status: string(guard.DeliveryDelivered)}
	code, err := f.d.Deliver(ctx, sub, eventType, body)
	res.ResponseCode = code
	failure := ""
	if err != nil {
		failure = err.Error()
	} else if code >= 300 {
		failure = fmt.Sprintf("endpoint answered %d", code)
	}
	if failure != "" {
		res.Status = string(guard.DeliveryFailed)
		if qerr := f.q.Enqueue(ctx, queue.Job{DeliveryID: id, URL: sub.TargetURL}); qerr != nil {
			return res, qerr
		}
	}
	if err := f.st.MarkResult(id, res.Status, code, failure); err != nil {
		return res, err
	}
	return res, nil
}

func (f *Fanout) Resend(ctx context.Context, d *store.Delivery) (Result, error) {
	sub, err := f.st.GetSubscription(d.TenantID, d.SubscriptionID)
	if err != nil {
		return Result{}, err
	}
	code, err := f.d.Deliver(ctx, sub, d.EventType, []byte(d.Payload))
	res := Result{SubscriptionID: sub.ID, DeliveryID: d.ID, Status: string(guard.DeliveryDelivered), ResponseCode: code}
	failure := ""
	if err != nil {
		failure = err.Error()
		res.Status = string(guard.DeliveryFailed)
	}
	if merr := f.st.MarkResult(d.ID, res.Status, code, failure); merr != nil {
		return res, merr
	}
	return res, nil
}

func (f *Fanout) Ping(ctx context.Context, sub *store.Subscription) (int, error) {
	body, err := BuildBody(guard.PingTemplate, sub, "ping", nil)
	if err != nil {
		return 0, err
	}
	return f.d.Deliver(ctx, sub, "ping", body)
}

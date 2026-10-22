package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hookhub/internal/guard"
	"hookhub/internal/queue"
	"hookhub/internal/store"
)

type Worker struct {
	st *store.Store
	d  *Dispatcher
	q  *queue.Queue
}

func NewWorker(st *store.Store, d *Dispatcher, q *queue.Queue) *Worker {
	return &Worker{st: st, d: d, q: q}
}

func (w *Worker) Drain(ctx context.Context, limit int) (int, error) {
	handled := 0
	for handled < limit {
		job, err := w.q.Pop(ctx)
		if err != nil {
			return handled, err
		}
		if job == nil {
			break
		}
		if err := w.handle(ctx, job); err != nil {
			return handled, err
		}
		handled++
	}
	return handled, nil
}

func (w *Worker) handle(ctx context.Context, job *queue.Job) error {
	delivery, err := w.st.DeliveryByNumericID(job.DeliveryID)
	if err != nil {
		return fmt.Errorf("load delivery %d: %w", job.DeliveryID, err)
	}
	notice, err := json.Marshal(map[string]any{
		"delivery_id": delivery.ID,
		"event":       delivery.EventType,
		"payload":     json.RawMessage(delivery.Payload),
	})
	if err != nil {
		return err
	}
	if err := w.d.Notify(ctx, job.URL, notice); err != nil {
		return w.retryOrDisable(ctx, delivery, job, err)
	}
	if err := w.st.MarkResult(delivery.ID, string(guard.DeliveryDelivered), 200, ""); err != nil {
		return err
	}
	return w.acknowledge(ctx, delivery)
}

func (w *Worker) acknowledge(ctx context.Context, delivery *store.Delivery) error {
	ackURL, err := w.q.AckURL(ctx, delivery.ID)
	if err != nil || ackURL == "" {
		return err
	}
	return w.d.Notify(ctx, ackURL, []byte(`{"status":"delivered"}`))
}

func (w *Worker) retryOrDisable(ctx context.Context, delivery *store.Delivery, job *queue.Job, cause error) error {
	if job.Attempt+1 >= guard.MaxDeliveryAttempts {
		if err := w.st.SetStatus(delivery.SubscriptionID, string(guard.StatusDisabled)); err != nil {
			return err
		}
		return w.st.MarkResult(delivery.ID, string(guard.DeliveryFailed), 0, cause.Error())
	}
	time.Sleep(time.Duration(job.Attempt+1) * time.Second)
	return w.q.Enqueue(ctx, queue.Job{DeliveryID: job.DeliveryID, URL: job.URL, Attempt: job.Attempt + 1})
}

func (w *Worker) Heartbeat(ctx context.Context, handled int) error {
	body := []byte(fmt.Sprintf(`{"handled":%d}`, handled))
	return w.d.Notify(ctx, guard.OpsWebhookURL, body)
}

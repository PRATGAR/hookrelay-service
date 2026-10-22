package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	pendingKey = "retry:pending"
	jobTTL     = 24 * time.Hour
)

type Job struct {
	DeliveryID int64  `json:"delivery_id"`
	URL        string `json:"url"`
	Attempt    int    `json:"attempt"`
}

type Queue struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

func jobKey(deliveryID int64) string {
	return fmt.Sprintf("retry:job:%d", deliveryID)
}

func ackKey(deliveryID int64) string {
	return fmt.Sprintf("retry:ack:%d", deliveryID)
}

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if err := q.rdb.Set(ctx, jobKey(job.DeliveryID), raw, jobTTL).Err(); err != nil {
		return err
	}
	return q.rdb.SAdd(ctx, pendingKey, job.DeliveryID).Err()
}

func (q *Queue) Pop(ctx context.Context) (*Job, error) {
	id, err := q.rdb.SPop(ctx, pendingKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	key := "retry:job:" + id
	raw, err := q.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	if err := q.rdb.Del(ctx, key).Err(); err != nil {
		return nil, err
	}
	var job Job
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (q *Queue) SetAckURL(ctx context.Context, deliveryID int64, target string) error {
	return q.rdb.Set(ctx, ackKey(deliveryID), target, jobTTL).Err()
}

func (q *Queue) AckURL(ctx context.Context, deliveryID int64) (string, error) {
	v, err := q.rdb.Get(ctx, ackKey(deliveryID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}

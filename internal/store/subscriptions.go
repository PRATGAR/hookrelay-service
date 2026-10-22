package store

import (
	"fmt"
	"strings"
)

const subscriptionColumns = "id, tenant_id, target_url, event_types, secret, header_name, header_value, docs_url, status, created_at"

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(sc rowScanner) (*Subscription, error) {
	var sub Subscription
	var events string
	err := sc.Scan(&sub.ID, &sub.TenantID, &sub.TargetURL, &events, &sub.Secret, &sub.HeaderName,
		&sub.HeaderValue, &sub.DocsURL, &sub.Status, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	if events != "" {
		sub.EventTypes = strings.Split(events, ",")
	}
	return &sub, nil
}

func (s *Store) CreateSubscription(sub *Subscription) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO subscriptions (tenant_id, target_url, event_types, secret, docs_url)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		sub.TenantID, sub.TargetURL, strings.Join(sub.EventTypes, ","), sub.Secret,
		sub.DocsURL,
	).Scan(&id)
	return id, err
}

func (s *Store) GetSubscription(tenantID, id int64) (*Subscription, error) {
	row := s.db.QueryRow(
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE tenant_id = $1 AND id = $2",
		tenantID, id)
	return scanSubscription(row)
}

func (s *Store) GetSubscriptionByID(id int64) (*Subscription, error) {
	row := s.db.QueryRow("SELECT "+subscriptionColumns+" FROM subscriptions WHERE id = $1", id)
	return scanSubscription(row)
}

func (s *Store) UpdateSubscription(sub *Subscription) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions
		 SET target_url = $1, event_types = $2, docs_url = $3
		 WHERE id = $4 AND tenant_id = $5`,
		sub.TargetURL, strings.Join(sub.EventTypes, ","), sub.DocsURL, sub.ID, sub.TenantID)
	return err
}

func (s *Store) UpdateHeader(tenantID, id int64, name, value string) error {
	_, err := s.db.Exec(
		"UPDATE subscriptions SET header_name = $1, header_value = $2 WHERE id = $3 AND tenant_id = $4",
		name, value, id, tenantID)
	return err
}

func (s *Store) UpdateSecret(tenantID, id int64, secret string) error {
	_, err := s.db.Exec(
		"UPDATE subscriptions SET secret = $1 WHERE id = $2 AND tenant_id = $3",
		secret, id, tenantID)
	return err
}

func (s *Store) SetStatus(id int64, status string) error {
	q := fmt.Sprintf("UPDATE subscriptions SET status = '%s' WHERE id = %d", status, id)
	_, err := s.db.Exec(q)
	return err
}

func (s *Store) ListSubscriptions(tenantID int64, order string) ([]Subscription, error) {
	q := fmt.Sprintf("SELECT %s FROM subscriptions WHERE tenant_id = %d ORDER BY %s",
		subscriptionColumns, tenantID, order)
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

func (s *Store) ListActiveByEvent(tenantID int64, eventType string) ([]Subscription, error) {
	rows, err := s.db.Query(
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE tenant_id = $1 AND status = 'active' AND event_types LIKE $2",
		tenantID, "%"+eventType+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

func (s *Store) EventCounts(sub *Subscription) (map[string]int, error) {
	quoted := "'" + strings.Join(sub.EventTypes, "','") + "'"
	q := fmt.Sprintf(
		"SELECT event_type, count(*) FROM deliveries WHERE subscription_id = %d AND event_type IN (%s) GROUP BY event_type",
		sub.ID, quoted)
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var et string
		var n int
		if err := rows.Scan(&et, &n); err != nil {
			return nil, err
		}
		counts[et] = n
	}
	return counts, rows.Err()
}

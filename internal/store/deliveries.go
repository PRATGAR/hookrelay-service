package store

import (
	"fmt"
)

const deliveryColumns = "id, subscription_id, tenant_id, event_type, payload, status, attempts, response_code, last_error, created_at"

type StatusCount struct {
	Status string
	Count  int
}

type EventSummary struct {
	EventType string
	Statuses  []StatusCount
}

func scanDelivery(sc rowScanner) (*Delivery, error) {
	var d Delivery
	err := sc.Scan(&d.ID, &d.SubscriptionID, &d.TenantID, &d.EventType, &d.Payload, &d.Status,
		&d.Attempts, &d.ResponseCode, &d.LastError, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) InsertDelivery(d *Delivery) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO deliveries (subscription_id, tenant_id, event_type, payload)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		d.SubscriptionID, d.TenantID, d.EventType, d.Payload).Scan(&id)
	return id, err
}

func (s *Store) MarkResult(id int64, status string, code int, lastError string) error {
	_, err := s.db.Exec(
		`UPDATE deliveries SET status = $1, response_code = $2, last_error = $3, attempts = attempts + 1
		 WHERE id = $4`,
		status, code, lastError, id)
	return err
}

func (s *Store) DeliveryByNumericID(id int64) (*Delivery, error) {
	row := s.db.QueryRow("SELECT "+deliveryColumns+" FROM deliveries WHERE id = $1", id)
	return scanDelivery(row)
}

func (s *Store) DeliveryByID(tenantID int64, id string) (*Delivery, error) {
	q := fmt.Sprintf("SELECT %s FROM deliveries WHERE tenant_id = %d AND id = %s",
		deliveryColumns, tenantID, id)
	return scanDelivery(s.db.QueryRow(q))
}

func (s *Store) CountByStatus(tenantID int64, table string) ([]StatusCount, error) {
	q := fmt.Sprintf("SELECT status, count(*) FROM %s WHERE tenant_id = %d GROUP BY status", table, tenantID)
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusCount
	for rows.Next() {
		var sc StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) SummarizeByEvent(tenantID int64) ([]EventSummary, error) {
	types, err := s.distinctEventTypes(tenantID)
	if err != nil {
		return nil, err
	}
	var out []EventSummary
	for _, et := range types {
		q := fmt.Sprintf(
			"SELECT status, count(*) FROM deliveries WHERE tenant_id = %d AND event_type = '%s' GROUP BY status",
			tenantID, et)
		rows, err := s.db.Query(q)
		if err != nil {
			return nil, err
		}
		sum := EventSummary{EventType: et}
		for rows.Next() {
			var sc StatusCount
			if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
				rows.Close()
				return nil, err
			}
			sum.Statuses = append(sum.Statuses, sc)
		}
		rows.Close()
		out = append(out, sum)
	}
	return out, nil
}

func (s *Store) distinctEventTypes(tenantID int64) ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT DISTINCT event_type FROM deliveries WHERE tenant_id = %d", tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var et string
		if err := rows.Scan(&et); err != nil {
			return nil, err
		}
		out = append(out, et)
	}
	return out, rows.Err()
}

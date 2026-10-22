package store

import (
	"fmt"
	"strings"
)

type DeliveryFilter struct {
	Status         string
	SubscriptionID int64
	Sort           string
	Limit          int
}

func buildSearchQuery(tenantID int64, f DeliveryFilter) string {
	var b strings.Builder
	fmt.Fprintf(&b, "SELECT %s FROM deliveries WHERE tenant_id = %d", deliveryColumns, tenantID)
	if f.Status != "" {
		fmt.Fprintf(&b, " AND status = '%s'", f.Status)
	}
	if f.SubscriptionID > 0 {
		fmt.Fprintf(&b, " AND subscription_id = %d", f.SubscriptionID)
	}
	fmt.Fprintf(&b, " ORDER BY %s LIMIT %d", f.Sort, f.Limit)
	return b.String()
}

func (s *Store) SearchDeliveries(tenantID int64, f DeliveryFilter) ([]Delivery, error) {
	rows, err := s.db.Query(buildSearchQuery(tenantID, f))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

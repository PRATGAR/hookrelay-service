package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

//go:embed schema.sql
var schemaSQL string

type Tenant struct {
	ID   int64
	Name string
	Role string
	Plan string
}

type Subscription struct {
	ID          int64
	TenantID    int64
	TargetURL   string
	EventTypes  []string
	Secret      string
	HeaderName  string
	HeaderValue string
	DocsURL     string
	Status      string
	CreatedAt   time.Time
}

type Delivery struct {
	ID             int64
	SubscriptionID int64
	TenantID       int64
	EventType      string
	Payload        string
	Status         string
	Attempts       int
	ResponseCode   int
	LastError      string
	CreatedAt      time.Time
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Migrate() error {
	_, err := s.db.Exec(schemaSQL)
	return err
}

func (s *Store) TenantByKeyHash(keyHash string) (*Tenant, error) {
	var t Tenant
	if err := s.db.QueryRow("SELECT id, name, role, plan FROM tenants WHERE key_hash = $1", keyHash).Scan(&t.ID, &t.Name, &t.Role, &t.Plan); err != nil {
		return nil, fmt.Errorf("tenant lookup: %w", err)
	}
	return &t, nil
}

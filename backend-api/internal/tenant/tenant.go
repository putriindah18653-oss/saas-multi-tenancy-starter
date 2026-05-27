package tenant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}
type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }
func slug(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}
func (s *Service) List(ctx context.Context) ([]Tenant, error) {
	rows, err := s.db.Query(ctx, "SELECT id,name,slug,status FROM tenants WHERE status<>'deleted' ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tenant{}
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Service) Create(ctx context.Context, creatorUserID, name, sl string) (Tenant, error) {
	if creatorUserID == "" {
		return Tenant{}, errors.New("creator user is required")
	}
	if sl == "" {
		sl = slug(name)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Tenant{}, err
	}
	defer tx.Rollback(ctx)
	var t Tenant
	err = tx.QueryRow(ctx, "INSERT INTO tenants(name,slug,status) VALUES($1,$2,'active') RETURNING id,name,slug,status", name, sl).Scan(&t.ID, &t.Name, &t.Slug, &t.Status)
	if err != nil {
		return Tenant{}, err
	}
	if creatorUserID != "" {
		if _, err := tx.Exec(ctx, "INSERT INTO user_tenants(user_id,tenant_id,role,is_active) VALUES($1,$2,'owner-tenant',true) ON CONFLICT(user_id,tenant_id) DO UPDATE SET role='owner-tenant',is_active=true,updated_at=now()", creatorUserID, t.ID); err != nil {
			return Tenant{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, err
	}
	return t, nil
}
func (s *Service) Get(ctx context.Context, id string) (Tenant, error) {
	var t Tenant
	err := s.db.QueryRow(ctx, "SELECT id,name,slug,status FROM tenants WHERE id=$1 AND status<>'deleted'", id).Scan(&t.ID, &t.Name, &t.Slug, &t.Status)
	return t, err
}
func (s *Service) Update(ctx context.Context, id, name, status string) (Tenant, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return cur, err
	}
	if name == "" {
		name = cur.Name
	}
	if status == "" {
		status = cur.Status
	}
	var t Tenant
	err = s.db.QueryRow(ctx, "UPDATE tenants SET name=$2,status=$3,updated_at=now() WHERE id=$1 RETURNING id,name,slug,status", id, name, status).Scan(&t.ID, &t.Name, &t.Slug, &t.Status)
	return t, err
}
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "UPDATE tenants SET status='deleted',updated_at=now() WHERE id=$1", id)
	return err
}

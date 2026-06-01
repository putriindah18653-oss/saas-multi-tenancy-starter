package tenant

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/database"
)

type Tenant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}
type Service struct{ db *pgxpool.Pool }

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])?$`)

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }
func slug(s string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == ' ', r == '-', r == '_':
			return '-'
		default:
			return -1
		}
	}, strings.TrimSpace(s)), "-")
}
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return "", errors.New("tenant name must be 1-120 characters")
	}
	return name, nil
}
func validateSlug(sl string) (string, error) {
	sl = strings.TrimSpace(strings.ToLower(sl))
	if !slugPattern.MatchString(sl) {
		return "", errors.New("tenant slug must be 3-64 chars using lowercase letters, numbers, and hyphens")
	}
	return sl, nil
}
func validateStatus(status string) (string, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "active", "suspended":
		return status, nil
	case "deleted":
		return "", errors.New("deleted status requires delete endpoint")
	default:
		return "", errors.New("invalid tenant status")
	}
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
	var err error
	name, err = validateName(name)
	if err != nil {
		return Tenant{}, err
	}
	if sl == "" {
		sl = slug(name)
	}
	sl, err = validateSlug(sl)
	if err != nil {
		return Tenant{}, err
	}
	var t Tenant
	err = database.WithRLS(ctx, s.db, database.RLSContext{UserID: creatorUserID, PlatformAdmin: true}, func(q database.Querier) error {
		if err := q.QueryRow(ctx, "INSERT INTO tenants(name,slug,status) VALUES($1,$2,'active') RETURNING id,name,slug,status", name, sl).Scan(&t.ID, &t.Name, &t.Slug, &t.Status); err != nil {
			return err
		}
		// creatorUserID is always non-empty at this point (validated above).
		if _, err := q.Exec(ctx, "INSERT INTO user_tenants(user_id,tenant_id,role,is_active) VALUES($1,$2,'owner-tenant',true) ON CONFLICT(user_id,tenant_id) DO UPDATE SET role='owner-tenant',is_active=true,updated_at=now()", creatorUserID, t.ID); err != nil {
			return err
		}
		_, err := q.Exec(ctx, "INSERT INTO tenant_settings(tenant_id,display_name) VALUES($1,$2) ON CONFLICT(tenant_id) DO NOTHING", t.ID, t.Name)
		return err
	})
	return t, err
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
	if strings.TrimSpace(name) == "" {
		name = cur.Name
	}
	name, err = validateName(name)
	if err != nil {
		return Tenant{}, err
	}
	if strings.TrimSpace(status) == "" {
		status = cur.Status
	}
	status, err = validateStatus(status)
	if err != nil {
		return Tenant{}, err
	}
	var t Tenant
	err = s.db.QueryRow(ctx, "UPDATE tenants SET name=$2,status=$3,updated_at=now() WHERE id=$1 RETURNING id,name,slug,status", id, name, status).Scan(&t.ID, &t.Name, &t.Slug, &t.Status)
	return t, err
}
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "UPDATE tenants SET status='deleted',updated_at=now() WHERE id=$1", id)
	return err
}

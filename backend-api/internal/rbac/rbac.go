package rbac

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }
func (s *Service) HasAppRole(role, required string) bool {
	return role == "owner-app" || role == required
}
func (s *Service) AppPermissions(ctx context.Context, appRole string) ([]string, error) {
	if appRole == "" {
		return []string{}, nil
	}
	rows, err := s.db.Query(ctx, "SELECT p.key FROM roles r JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE r.scope='app' AND r.name=$1 ORDER BY p.key", appRole)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
func (s *Service) TenantPermissions(ctx context.Context, role string) ([]string, error) {
	rows, err := s.db.Query(ctx, "SELECT p.key FROM roles r JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE r.scope='tenant' AND r.name=$1 ORDER BY p.key", role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
func (s *Service) HasPermission(ctx context.Context, appRole, tenantRole, key string) (bool, error) {
	role := appRole
	scope := "app"
	if len(key) > 7 && key[:7] == "tenant." {
		role = tenantRole
		scope = "tenant"
	} else if appRole == "owner-app" {
		return true, nil
	}
	var ok bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM roles r JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE r.scope=$1 AND r.name=$2 AND p.key=$3)", scope, role, key).Scan(&ok)
	if err != nil || ok || scope != "app" || appRole != "admin" {
		return ok, err
	}

	// Backward compatibility for databases migrated before app.settings permissions existed.
	// Keeps frontend-owner Settings usable until latest migrations are applied.
	switch key {
	case "app.settings.read", "app.settings.update":
		return true, nil
	default:
		return false, nil
	}
}
func (s *Service) TenantMembership(ctx context.Context, userID, tenantID string) (role string, ok bool, err error) {
	err = s.db.QueryRow(ctx, "SELECT role FROM user_tenants WHERE user_id=$1 AND tenant_id=$2 AND is_active=true", userID, tenantID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return role, true, nil
}

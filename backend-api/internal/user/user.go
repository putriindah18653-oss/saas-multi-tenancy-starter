package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Member struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}
type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }
func tempPassword() string                 { b := make([]byte, 8); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func ValidTenantRole(role string) bool {
	switch role {
	case "owner-tenant", "admin", "finance", "support":
		return true
	default:
		return false
	}
}
func hash(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}
func (s *Service) TenantMe(ctx context.Context, userID, tenantID string) (Member, error) {
	var m Member
	err := s.db.QueryRow(ctx, "SELECT ut.id,u.id,u.name,u.email,ut.tenant_id,ut.role,ut.is_active FROM user_tenants ut JOIN users u ON u.id=ut.user_id WHERE ut.user_id=$1 AND ut.tenant_id=$2", userID, tenantID).Scan(&m.ID, &m.UserID, &m.Name, &m.Email, &m.TenantID, &m.Role, &m.IsActive)
	return m, err
}
func (s *Service) List(ctx context.Context, tenantID string) ([]Member, error) {
	rows, err := s.db.Query(ctx, "SELECT ut.id,u.id,u.name,u.email,ut.tenant_id,ut.role,ut.is_active FROM user_tenants ut JOIN users u ON u.id=ut.user_id WHERE ut.tenant_id=$1 ORDER BY u.name", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.Email, &m.TenantID, &m.Role, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Service) Invite(ctx context.Context, tenantID, name, email, role string) (Member, string, error) {
	if !ValidTenantRole(role) {
		return Member{}, "", errors.New("invalid role")
	}
	pass := tempPassword()
	h, err := hash(pass)
	if err != nil {
		return Member{}, "", err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Member{}, "", err
	}
	defer tx.Rollback(ctx)
	var userID string
	createdUser := false
	err = tx.QueryRow(ctx, "INSERT INTO users(name,email,password_hash) VALUES($1,$2,$3) ON CONFLICT(email) DO NOTHING RETURNING id", name, email, h).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, "SELECT id FROM users WHERE email=$1", email).Scan(&userID)
	} else if err == nil {
		createdUser = true
	}
	if err != nil {
		return Member{}, "", err
	}
	var m Member
	err = tx.QueryRow(ctx, "INSERT INTO user_tenants(user_id,tenant_id,role) VALUES($1,$2,$3) ON CONFLICT(user_id,tenant_id) DO UPDATE SET role=$3,is_active=true,updated_at=now() RETURNING id,user_id,tenant_id,role,is_active", userID, tenantID, role).Scan(&m.ID, &m.UserID, &m.TenantID, &m.Role, &m.IsActive)
	if err != nil {
		return Member{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Member{}, "", err
	}
	m.Name = name
	m.Email = email
	if !createdUser {
		pass = ""
	}
	return m, pass, nil
}
func (s *Service) ChangeRole(ctx context.Context, tenantID, memberID, role string) (Member, error) {
	if !ValidTenantRole(role) {
		return Member{}, errors.New("invalid role")
	}
	var m Member
	err := s.db.QueryRow(ctx, "UPDATE user_tenants ut SET role=$3,updated_at=now() FROM users u WHERE ut.user_id=u.id AND ut.tenant_id=$1 AND ut.id=$2 RETURNING ut.id,u.id,u.name,u.email,ut.tenant_id,ut.role,ut.is_active", tenantID, memberID, role).Scan(&m.ID, &m.UserID, &m.Name, &m.Email, &m.TenantID, &m.Role, &m.IsActive)
	return m, err
}
func (s *Service) Remove(ctx context.Context, tenantID, memberID string) error {
	tag, err := s.db.Exec(ctx, "UPDATE user_tenants SET is_active=false,updated_at=now() WHERE tenant_id=$1 AND id=$2", tenantID, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

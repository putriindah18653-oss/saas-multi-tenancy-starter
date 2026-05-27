package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend/internal/config"
)

type Context struct{ UserID, Email, AppRole string }
type TenantMembership struct {
	ID, TenantID, TenantName, TenantSlug, Role string
	IsActive                                   bool
}
type UserProfile struct {
	ID, Name, Email, AppRole string
	IsActive                 bool
	Tenants                  []TenantMembership
}
type Claims struct {
	UserID, Email, AppRole, TokenType string
	jwt.RegisteredClaims
}
type Service struct {
	db  *pgxpool.Pool
	cfg config.JWTConfig
}

func NewService(db *pgxpool.Pool, cfg config.JWTConfig) *Service { return &Service{db: db, cfg: cfg} }
func HashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}
func VerifyPassword(hash, p string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p))
}

func (s *Service) RegisterOwner(ctx context.Context, name, email, password string) (*UserProfile, string, string, error) {
	var exists bool
	if err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE app_role='owner-app')").Scan(&exists); err != nil {
		return nil, "", "", err
	}
	if exists {
		return nil, "", "", errors.New("owner-app already exists")
	}
	h, err := HashPassword(password)
	if err != nil {
		return nil, "", "", err
	}
	var u UserProfile
	err = s.db.QueryRow(ctx, "INSERT INTO users(name,email,password_hash,app_role) VALUES($1,$2,$3,'owner-app') RETURNING id,name,email,COALESCE(app_role,''),is_active", name, email, h).Scan(&u.ID, &u.Name, &u.Email, &u.AppRole, &u.IsActive)
	if err != nil {
		return nil, "", "", err
	}
	a, r, err := s.Tokens(&u)
	return &u, a, r, err
}
func (s *Service) Login(ctx context.Context, email, password string) (*UserProfile, string, string, error) {
	var hash string
	u := UserProfile{}
	err := s.db.QueryRow(ctx, "SELECT id,name,email,password_hash,COALESCE(app_role,''),is_active FROM users WHERE email=$1", email).Scan(&u.ID, &u.Name, &u.Email, &hash, &u.AppRole, &u.IsActive)
	if err != nil {
		return nil, "", "", err
	}
	if !u.IsActive {
		return nil, "", "", errors.New("user inactive")
	}
	if err := VerifyPassword(hash, password); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}
	u.Tenants, _ = s.Memberships(ctx, u.ID)
	a, r, err := s.Tokens(&u)
	return &u, a, r, err
}
func (s *Service) Me(ctx context.Context, userID string) (*UserProfile, error) {
	u := UserProfile{}
	err := s.db.QueryRow(ctx, "SELECT id,name,email,COALESCE(app_role,''),is_active FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.AppRole, &u.IsActive)
	if err != nil {
		return nil, err
	}
	u.Tenants, _ = s.Memberships(ctx, u.ID)
	return &u, nil
}
func (s *Service) Memberships(ctx context.Context, userID string) ([]TenantMembership, error) {
	rows, err := s.db.Query(ctx, "SELECT ut.id,t.id,t.name,t.slug,ut.role,ut.is_active FROM user_tenants ut JOIN tenants t ON t.id=ut.tenant_id WHERE ut.user_id=$1 ORDER BY t.name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TenantMembership{}
	for rows.Next() {
		var m TenantMembership
		if err := rows.Scan(&m.ID, &m.TenantID, &m.TenantName, &m.TenantSlug, &m.Role, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Service) Tokens(u *UserProfile) (string, string, error) {
	a, err := s.token(u, "access", time.Duration(s.cfg.AccessTTLMinutes)*time.Minute)
	if err != nil {
		return "", "", err
	}
	r, err := s.token(u, "refresh", time.Duration(s.cfg.RefreshTTLHours)*time.Hour)
	return a, r, err
}
func (s *Service) token(u *UserProfile, typ string, ttl time.Duration) (string, error) {
	secret := s.cfg.AccessSecret
	if secret == "" {
		secret = "change-me-access-secret"
	}
	if typ == "refresh" {
		secret = s.cfg.RefreshSecret
		if secret == "" {
			secret = "change-me-refresh-secret"
		}
	}
	claims := Claims{UserID: u.ID, Email: u.Email, AppRole: u.AppRole, TokenType: typ, RegisteredClaims: jwt.RegisteredClaims{Subject: u.ID, ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
func (s *Service) Parse(tokenString, typ string) (*Claims, error) {
	secret := s.cfg.AccessSecret
	if secret == "" {
		secret = "change-me-access-secret"
	}
	if typ == "refresh" {
		secret = s.cfg.RefreshSecret
		if secret == "" {
			secret = "change-me-refresh-secret"
		}
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) { return []byte(secret), nil })
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != typ {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

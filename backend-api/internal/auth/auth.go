package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
)

type Context struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	AppRole string `json:"app_role"`
}
type TenantMembership struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Role       string `json:"role"`
	IsActive   bool   `json:"is_active"`
}
type UserProfile struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Email              string             `json:"email"`
	AppRole            string             `json:"app_role"`
	IsActive           bool               `json:"is_active"`
	MustChangePassword bool               `json:"must_change_password"`
	Tenants            []TenantMembership `json:"tenant_memberships"`
}
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	AppRole   string `json:"app_role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}
type Session struct {
	ID        string     `json:"id"`
	UserAgent string     `json:"user_agent"`
	IPAddress string     `json:"ip_address"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
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
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *Service) Login(ctx context.Context, email, password, ip, ua string) (*UserProfile, string, string, error) {
	var hash string
	u := UserProfile{}
	err := s.db.QueryRow(ctx, "SELECT id,name,email,password_hash,COALESCE(app_role,''),is_active,must_change_password FROM users WHERE email=$1", email).Scan(&u.ID, &u.Name, &u.Email, &hash, &u.AppRole, &u.IsActive, &u.MustChangePassword)
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
	a, r, err := s.Tokens(ctx, &u, ip, ua)
	return &u, a, r, err
}
func (s *Service) Me(ctx context.Context, userID string) (*UserProfile, error) {
	u := UserProfile{}
	err := s.db.QueryRow(ctx, "SELECT id,name,email,COALESCE(app_role,''),is_active,must_change_password FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.AppRole, &u.IsActive, &u.MustChangePassword)
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
func (s *Service) Tokens(ctx context.Context, u *UserProfile, ip, ua string) (string, string, error) {
	a, err := s.token(u, "access", "", time.Duration(s.cfg.AccessTTLMinutes)*time.Minute)
	if err != nil {
		return "", "", err
	}
	jti, err := randomHex(16)
	if err != nil {
		return "", "", err
	}
	expires := time.Now().Add(time.Duration(s.cfg.RefreshTTLHours) * time.Hour)
	r, err := s.token(u, "refresh", jti, time.Until(expires))
	if err != nil {
		return "", "", err
	}
	_, err = s.db.Exec(ctx, "INSERT INTO refresh_tokens(user_id,token_hash,jti,user_agent,ip_address,expires_at) VALUES($1,$2,$3,$4,$5,$6)", u.ID, tokenHash(r), jti, ua, ip, expires)
	if err != nil {
		return "", "", err
	}
	return a, r, nil
}
func (s *Service) token(u *UserProfile, typ, jti string, ttl time.Duration) (string, error) {
	secret, err := s.secret(typ)
	if err != nil {
		return "", err
	}
	claims := Claims{UserID: u.ID, Email: u.Email, AppRole: u.AppRole, TokenType: typ, RegisteredClaims: jwt.RegisteredClaims{Subject: u.ID, ID: jti, ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
func (s *Service) secret(typ string) (string, error) {
	secret := s.cfg.AccessSecret
	if typ == "refresh" {
		secret = s.cfg.RefreshSecret
	}
	if secret == "" {
		return "", errors.New("jwt secret not configured")
	}
	return secret, nil
}
func (s *Service) Parse(tokenString, typ string) (*Claims, error) {
	secret, err := s.secret(typ)
	if err != nil {
		return nil, errors.New("invalid token")
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
func (s *Service) Refresh(ctx context.Context, refreshToken, ip, ua string) (*UserProfile, string, string, error) {
	c, err := s.Parse(refreshToken, "refresh")
	if err != nil {
		return nil, "", "", err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, "", "", err
	}
	defer tx.Rollback(ctx)

	var oldID string
	var revokedAt *time.Time
	var replacedBy *string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, "SELECT id,revoked_at,replaced_by,expires_at FROM refresh_tokens WHERE user_id=$1 AND jti=$2 AND token_hash=$3 FOR UPDATE", c.UserID, c.ID, tokenHash(refreshToken)).Scan(&oldID, &revokedAt, &replacedBy, &expiresAt)
	if err != nil {
		return nil, "", "", errors.New("invalid refresh token")
	}
	if revokedAt != nil || replacedBy != nil {
		_, _ = tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at, now()) WHERE user_id=$1 AND revoked_at IS NULL", c.UserID)
		_ = tx.Commit(ctx)
		return nil, "", "", errors.New("refresh token reuse detected")
	}
	if time.Now().After(expiresAt) {
		return nil, "", "", errors.New("invalid refresh token")
	}

	u, err := s.Me(ctx, c.UserID)
	if err != nil {
		return nil, "", "", err
	}
	a, err := s.token(u, "access", "", time.Duration(s.cfg.AccessTTLMinutes)*time.Minute)
	if err != nil {
		return nil, "", "", err
	}
	jti, err := randomHex(16)
	if err != nil {
		return nil, "", "", err
	}
	expires := time.Now().Add(time.Duration(s.cfg.RefreshTTLHours) * time.Hour)
	r, err := s.token(u, "refresh", jti, time.Until(expires))
	if err != nil {
		return nil, "", "", err
	}
	var newID string
	err = tx.QueryRow(ctx, "INSERT INTO refresh_tokens(user_id,token_hash,jti,user_agent,ip_address,expires_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id", u.ID, tokenHash(r), jti, ua, ip, expires).Scan(&newID)
	if err != nil {
		return nil, "", "", err
	}
	if _, err := tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at=now(),replaced_by=$2 WHERE id=$1 AND revoked_at IS NULL AND replaced_by IS NULL", oldID, newID); err != nil {
		return nil, "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, "", "", err
	}
	return u, a, r, nil
}
func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, "UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL", tokenHash(refreshToken))
	return err
}
func (s *Service) Sessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.db.Query(ctx, "SELECT id,user_agent,ip_address,expires_at,revoked_at,created_at FROM refresh_tokens WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	tag, err := s.db.Exec(ctx, "UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND id=$2 AND revoked_at IS NULL", userID, sessionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password too short")
	}
	var hash string
	if err := s.db.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1", userID).Scan(&hash); err != nil {
		return err
	}
	if err := VerifyPassword(hash, currentPassword); err != nil {
		return errors.New("invalid current password")
	}
	nh, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, "UPDATE users SET password_hash=$2,must_change_password=false,updated_at=now() WHERE id=$1", userID, nh)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(ctx, "UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL", userID)
	return nil
}
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

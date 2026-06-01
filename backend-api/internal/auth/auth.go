package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/database"
)

type Context struct {
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	AppRole           string `json:"app_role"`
	MustChangePassword bool   `json:"must_change_password"`
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
	Phone              string             `json:"phone"`
	Address            string             `json:"address"`
	AvatarURL          string             `json:"avatar_url"`
	Bio                string             `json:"bio"`
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
	err := s.db.QueryRow(ctx, "SELECT id,name,email,password_hash,COALESCE(phone,''),COALESCE(address,''),COALESCE(avatar_url,''),COALESCE(bio,''),COALESCE(app_role,''),is_active,must_change_password FROM users WHERE email=$1", email).Scan(&u.ID, &u.Name, &u.Email, &hash, &u.Phone, &u.Address, &u.AvatarURL, &u.Bio, &u.AppRole, &u.IsActive, &u.MustChangePassword)
	if err != nil {
		return nil, "", "", err
	}
	if !u.IsActive {
		return nil, "", "", errors.New("user inactive")
	}
	if err := VerifyPassword(hash, password); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}
	tenants, err := s.Memberships(ctx, u.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("login: memberships: %w", err)
	}
	u.Tenants = tenants
	a, r, err := s.Tokens(ctx, &u, ip, ua)
	return &u, a, r, err
}
func (s *Service) Me(ctx context.Context, userID string) (*UserProfile, error) {
	u := UserProfile{}
	err := s.db.QueryRow(ctx, "SELECT id,name,email,COALESCE(phone,''),COALESCE(address,''),COALESCE(avatar_url,''),COALESCE(bio,''),COALESCE(app_role,''),is_active,must_change_password FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Address, &u.AvatarURL, &u.Bio, &u.AppRole, &u.IsActive, &u.MustChangePassword)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, errors.New("user inactive")
	}
	tenants, err := s.Memberships(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("me: memberships: %w", err)
	}
	u.Tenants = tenants
	return &u, nil
}
func (s *Service) UpdateProfile(ctx context.Context, userID, name, phone, address, avatarURL, bio string) (*UserProfile, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	address = strings.TrimSpace(address)
	avatarURL = strings.TrimSpace(avatarURL)
	bio = strings.TrimSpace(bio)

	if name == "" {
		return nil, errors.New("name required")
	}
	if len([]rune(name)) > 120 {
		return nil, errors.New("name too long")
	}
	if len([]rune(phone)) > 40 {
		return nil, errors.New("phone too long")
	}
	if len([]rune(address)) > 500 {
		return nil, errors.New("address too long")
	}
	if len([]rune(avatarURL)) > 500 {
		return nil, errors.New("avatar url too long")
	}
	if len([]rune(bio)) > 500 {
		return nil, errors.New("bio too long")
	}

	tag, err := s.db.Exec(ctx, "UPDATE users SET name=$2,phone=$3,address=$4,avatar_url=$5,bio=$6,updated_at=now() WHERE id=$1", userID, name, phone, address, avatarURL, bio)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return s.Me(ctx, userID)
}
func (s *Service) Memberships(ctx context.Context, userID string) ([]TenantMembership, error) {
	out := []TenantMembership{}
	err := database.WithRLS(ctx, s.db, database.RLSContext{UserID: userID}, func(q database.Querier) error {
		rows, err := q.Query(ctx, "SELECT ut.id,t.id,t.name,t.slug,ut.role,ut.is_active FROM user_tenants ut JOIN tenants t ON t.id=ut.tenant_id WHERE ut.user_id=$1 AND ut.is_active=true AND t.status='active' ORDER BY t.name", userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m TenantMembership
			if err := rows.Scan(&m.ID, &m.TenantID, &m.TenantName, &m.TenantSlug, &m.Role, &m.IsActive); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
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
	claims := Claims{UserID: u.ID, Email: u.Email, AppRole: u.AppRole, TokenType: typ, RegisteredClaims: jwt.RegisteredClaims{Subject: u.ID, ID: jti, Issuer: "saas-starter-api", Audience: jwt.ClaimStrings{typ}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now())}}
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
	if claims.TokenType != typ || claims.Issuer != "saas-starter-api" {
		return nil, errors.New("invalid token type")
	}
	hasAudience := false
	for _, audience := range claims.Audience {
		if audience == typ {
			hasAudience = true
			break
		}
	}
	if !hasAudience {
		return nil, errors.New("invalid token audience")
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
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	var hash string
	if err := s.db.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1", userID).Scan(&hash); err != nil {
		return err
	}
	if err := VerifyPassword(hash, currentPassword); err != nil {
		return errors.New("invalid current password")
	}
	if err := VerifyPassword(hash, newPassword); err == nil {
		return errors.New("new password must be different")
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

func validatePassword(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	if len([]byte(password)) > 72 {
		return errors.New("password must be at most 72 bytes")
	}
	lower := strings.ToLower(password)
	weak := []string{"password", "qwerty", "123456", "admin", "letmein"}
	for _, token := range weak {
		if strings.Contains(lower, token) {
			return errors.New("password is too common")
		}
	}
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit || !hasSymbol {
		return errors.New("password must include lowercase, uppercase, number, and symbol")
	}
	return nil
}

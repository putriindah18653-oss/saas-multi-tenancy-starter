package tenant

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
)

type Settings struct {
	TenantID    string         `json:"tenant_id"`
	DisplayName string         `json:"display_name"`
	LogoURL     string         `json:"logo_url"`
	Timezone    string         `json:"timezone"`
	Locale      string         `json:"locale"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata"`
}

func (s Settings) validate() error {
	s.DisplayName = strings.TrimSpace(s.DisplayName)
	s.LogoURL = strings.TrimSpace(s.LogoURL)
	s.Timezone = strings.TrimSpace(s.Timezone)
	s.Locale = strings.TrimSpace(s.Locale)
	s.Currency = strings.TrimSpace(s.Currency)
	if s.DisplayName != "" && len(s.DisplayName) > 120 {
		return errors.New("display_name too long")
	}
	if s.LogoURL != "" {
		u, err := url.ParseRequestURI(s.LogoURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return errors.New("logo_url must be http or https url")
		}
		if len(s.LogoURL) > 500 {
			return errors.New("logo_url too long")
		}
	}
	if s.Timezone != "" {
		if _, err := time.LoadLocation(s.Timezone); err != nil {
			return errors.New("invalid timezone")
		}
	}
	if s.Locale != "" && (len(s.Locale) < 2 || len(s.Locale) > 20) {
		return errors.New("invalid locale")
	}
	if s.Currency != "" {
		if len(s.Currency) != 3 {
			return errors.New("currency must be ISO 4217 code")
		}
		for _, r := range s.Currency {
			if r < 'A' || r > 'Z' {
				return errors.New("currency must be uppercase ISO 4217 code")
			}
		}
	}
	if len(s.Metadata) > 50 {
		return errors.New("metadata has too many keys")
	}
	return nil
}

func (s *Service) Settings(ctx context.Context, tenantID string) (Settings, error) {
	_, _ = s.db.Exec(ctx, "INSERT INTO tenant_settings(tenant_id,display_name) SELECT id,name FROM tenants WHERE id=$1 ON CONFLICT(tenant_id) DO NOTHING", tenantID)
	var out Settings
	err := s.db.QueryRow(ctx, "SELECT tenant_id,display_name,logo_url,timezone,locale,currency,metadata FROM tenant_settings WHERE tenant_id=$1", tenantID).Scan(&out.TenantID, &out.DisplayName, &out.LogoURL, &out.Timezone, &out.Locale, &out.Currency, &out.Metadata)
	return out, err
}
func (s *Service) UpdateSettings(ctx context.Context, tenantID string, in Settings) (Settings, error) {
	cur, err := s.Settings(ctx, tenantID)
	if err != nil {
		return cur, err
	}
	if err := in.validate(); err != nil {
		return cur, err
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.LogoURL = strings.TrimSpace(in.LogoURL)
	in.Timezone = strings.TrimSpace(in.Timezone)
	in.Locale = strings.TrimSpace(in.Locale)
	in.Currency = strings.TrimSpace(in.Currency)
	if in.DisplayName == "" {
		in.DisplayName = cur.DisplayName
	}
	if in.LogoURL == "" {
		in.LogoURL = cur.LogoURL
	}
	if in.Timezone == "" {
		in.Timezone = cur.Timezone
	}
	if in.Locale == "" {
		in.Locale = cur.Locale
	}
	if in.Currency == "" {
		in.Currency = cur.Currency
	}
	if in.Metadata == nil {
		in.Metadata = cur.Metadata
	}
	var out Settings
	err = s.db.QueryRow(ctx, "UPDATE tenant_settings SET display_name=$2,logo_url=$3,timezone=$4,locale=$5,currency=$6,metadata=$7,updated_at=now() WHERE tenant_id=$1 RETURNING tenant_id,display_name,logo_url,timezone,locale,currency,metadata", tenantID, in.DisplayName, in.LogoURL, in.Timezone, in.Locale, in.Currency, in.Metadata).Scan(&out.TenantID, &out.DisplayName, &out.LogoURL, &out.Timezone, &out.Locale, &out.Currency, &out.Metadata)
	return out, err
}

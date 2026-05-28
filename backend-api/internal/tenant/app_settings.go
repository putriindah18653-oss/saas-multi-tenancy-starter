package tenant

import (
	"context"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

type AppSettings struct {
	DisplayName  string         `json:"display_name"`
	LegalName    string         `json:"legal_name"`
	LogoURL      string         `json:"logo_url"`
	WebsiteURL   string         `json:"website_url"`
	SupportEmail string         `json:"support_email"`
	SupportPhone string         `json:"support_phone"`
	Timezone     string         `json:"timezone"`
	Locale       string         `json:"locale"`
	Currency     string         `json:"currency"`
	Address      string         `json:"address"`
	Metadata     map[string]any `json:"metadata"`
}

func (s AppSettings) validate() error {
	if s.DisplayName != "" && len(s.DisplayName) > 120 {
		return errors.New("display_name too long")
	}
	if len(s.LegalName) > 160 {
		return errors.New("legal_name too long")
	}
	if err := validateHTTPURL(s.LogoURL, "logo_url"); err != nil {
		return err
	}
	if err := validateHTTPURL(s.WebsiteURL, "website_url"); err != nil {
		return err
	}
	if s.SupportEmail != "" {
		if _, err := mail.ParseAddress(s.SupportEmail); err != nil || len(s.SupportEmail) > 160 {
			return errors.New("support_email must be valid")
		}
	}
	if len(s.SupportPhone) > 40 {
		return errors.New("support_phone too long")
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
	if len(s.Address) > 500 {
		return errors.New("address too long")
	}
	if len(s.Metadata) > 50 {
		return errors.New("metadata has too many keys")
	}
	return nil
}

func validateHTTPURL(value, field string) error {
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "/api/") || strings.HasPrefix(value, "/uploads/") {
		if len(value) > 500 {
			return errors.New(field + " too long")
		}
		return nil
	}
	u, err := url.ParseRequestURI(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New(field + " must be http or https url")
	}
	if len(value) > 500 {
		return errors.New(field + " too long")
	}
	return nil
}

func (s *Service) AppSettings(ctx context.Context) (AppSettings, error) {
	_, _ = s.db.Exec(ctx, "INSERT INTO app_settings(id,display_name) VALUES(TRUE,'PortalOnline') ON CONFLICT(id) DO NOTHING")
	var out AppSettings
	err := s.db.QueryRow(ctx, "SELECT display_name,legal_name,logo_url,website_url,support_email,support_phone,timezone,locale,currency,address,metadata FROM app_settings WHERE id=TRUE").Scan(&out.DisplayName, &out.LegalName, &out.LogoURL, &out.WebsiteURL, &out.SupportEmail, &out.SupportPhone, &out.Timezone, &out.Locale, &out.Currency, &out.Address, &out.Metadata)
	return out, err
}

func (s *Service) UpdateAppSettings(ctx context.Context, in AppSettings) (AppSettings, error) {
	cur, err := s.AppSettings(ctx)
	if err != nil {
		return cur, err
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.LegalName = strings.TrimSpace(in.LegalName)
	in.LogoURL = strings.TrimSpace(in.LogoURL)
	in.WebsiteURL = strings.TrimSpace(in.WebsiteURL)
	in.SupportEmail = strings.ToLower(strings.TrimSpace(in.SupportEmail))
	in.SupportPhone = strings.TrimSpace(in.SupportPhone)
	in.Timezone = strings.TrimSpace(in.Timezone)
	in.Locale = strings.TrimSpace(in.Locale)
	in.Currency = strings.TrimSpace(in.Currency)
	in.Address = strings.TrimSpace(in.Address)
	if err := in.validate(); err != nil {
		return cur, err
	}
	if in.DisplayName == "" {
		in.DisplayName = cur.DisplayName
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
	var out AppSettings
	err = s.db.QueryRow(ctx, "UPDATE app_settings SET display_name=$1,legal_name=$2,logo_url=$3,website_url=$4,support_email=$5,support_phone=$6,timezone=$7,locale=$8,currency=$9,address=$10,metadata=$11,updated_at=now() WHERE id=TRUE RETURNING display_name,legal_name,logo_url,website_url,support_email,support_phone,timezone,locale,currency,address,metadata", in.DisplayName, in.LegalName, in.LogoURL, in.WebsiteURL, in.SupportEmail, in.SupportPhone, in.Timezone, in.Locale, in.Currency, in.Address, in.Metadata).Scan(&out.DisplayName, &out.LegalName, &out.LogoURL, &out.WebsiteURL, &out.SupportEmail, &out.SupportPhone, &out.Timezone, &out.Locale, &out.Currency, &out.Address, &out.Metadata)
	return out, err
}

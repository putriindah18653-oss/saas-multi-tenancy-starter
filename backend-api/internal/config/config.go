package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	CORS     CORSConfig
}

type AppConfig struct {
	Env                string `json:"env"`
	TrustProxy         bool   `json:"trust_proxy"`
	InternalProxySecret string `json:"internal_proxy_secret"`
}
type HTTPConfig struct {
	Port string `json:"port"`
	Addr string `json:"addr"`
}
type DatabaseConfig struct {
	URL                                             string
	MaxConns, MinConns                              int32
	MaxConnLifetime, MaxConnIdleTime, HealthTimeout time.Duration
}
type RedisConfig struct {
	Addr, Password                                        string
	DB                                                    int
	DialTimeout, ReadTimeout, WriteTimeout, HealthTimeout time.Duration
}
type JWTConfig struct {
	AccessSecret, RefreshSecret       string
	AccessTTLMinutes, RefreshTTLHours int
	RefreshCookieSecure               bool
	RefreshCookieSameSite             string
}
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
}

func Load() (*Config, error) {
	port := env("APP_PORT", "8080")
	cfg := &Config{
		App:  AppConfig{Env: env("APP_ENV", "development"), TrustProxy: envBool("TRUST_PROXY", false), InternalProxySecret: env("INTERNAL_PROXY_SECRET", "")},
		HTTP: HTTPConfig{Port: port, Addr: ":" + port},
		Database: DatabaseConfig{
			URL:      env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/saas_starter?sslmode=disable"),
			MaxConns: int32(envInt("DATABASE_MAX_CONNS", 10)), MinConns: int32(envInt("DATABASE_MIN_CONNS", 1)),
			MaxConnLifetime: envDuration("DATABASE_MAX_CONN_LIFETIME", time.Hour), MaxConnIdleTime: envDuration("DATABASE_MAX_CONN_IDLE_TIME", 30*time.Minute), HealthTimeout: envDuration("DATABASE_HEALTH_TIMEOUT", 3*time.Second),
		},
		Redis: RedisConfig{
			Addr: env("REDIS_ADDR", "localhost:6379"), Password: env("REDIS_PASSWORD", ""), DB: envInt("REDIS_DB", 0),
			DialTimeout: envDuration("REDIS_DIAL_TIMEOUT", 5*time.Second), ReadTimeout: envDuration("REDIS_READ_TIMEOUT", 3*time.Second), WriteTimeout: envDuration("REDIS_WRITE_TIMEOUT", 3*time.Second), HealthTimeout: envDuration("REDIS_HEALTH_TIMEOUT", 3*time.Second),
		},
		JWT: JWTConfig{
			AccessSecret:          env("JWT_ACCESS_SECRET", ""),
			RefreshSecret:         env("JWT_REFRESH_SECRET", ""),
			AccessTTLMinutes:      envInt("JWT_ACCESS_TTL_MINUTES", 15),
			RefreshTTLHours:       envInt("JWT_REFRESH_TTL_HOURS", 168),
			RefreshCookieSecure:   envBool("JWT_REFRESH_COOKIE_SECURE", false),
			RefreshCookieSameSite: strings.ToLower(env("JWT_REFRESH_COOKIE_SAME_SITE", "lax")),
		},
		CORS: CORSConfig{AllowedOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))},
	}
	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if !c.IsDevelopment() {
		if c.App.InternalProxySecret == "" {
			return fmt.Errorf("INTERNAL_PROXY_SECRET is required in production")
		}
		if len(c.JWT.AccessSecret) < 32 || len(c.JWT.RefreshSecret) < 32 {
			return fmt.Errorf("JWT secrets must be at least 32 characters in non-development environments")
		}
		if !c.JWT.RefreshCookieSecure {
			return fmt.Errorf("JWT_REFRESH_COOKIE_SECURE=true is required outside development")
		}
		if c.JWT.RefreshCookieSameSite == "none" && !c.JWT.RefreshCookieSecure {
			return fmt.Errorf("JWT_REFRESH_COOKIE_SECURE=true is required when JWT_REFRESH_COOKIE_SAME_SITE=none")
		}
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("CORS wildcard is not allowed outside development")
			}
		}
		if strings.Contains(c.JWT.AccessSecret, "change-me") || strings.Contains(c.JWT.RefreshSecret, "change-me") {
			return fmt.Errorf("default JWT secrets are not allowed outside development")
		}
	}
	if c.JWT.RefreshCookieSameSite != "lax" && c.JWT.RefreshCookieSameSite != "strict" && c.JWT.RefreshCookieSameSite != "none" {
		return fmt.Errorf("JWT_REFRESH_COOKIE_SAME_SITE must be one of: lax, strict, none")
	}
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development" || c.App.Env == "dev" || c.App.Env == "local"
}
func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return fallback
}
func envInt(key string, fallback int) int {
	v := env(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
func envBool(key string, fallback bool) bool {
	v := strings.ToLower(env(key, ""))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
func envDuration(key string, fallback time.Duration) time.Duration {
	v := env(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
func splitCSV(raw string) []string {
	out := []string{}
	for _, p := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

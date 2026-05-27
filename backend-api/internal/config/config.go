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
	Env        string `json:"env"`
	TrustProxy bool   `json:"trust_proxy"`
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
}
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
}

func Load() (*Config, error) {
	port := env("APP_PORT", "8080")
	cfg := &Config{
		App:  AppConfig{Env: env("APP_ENV", "development"), TrustProxy: envBool("TRUST_PROXY", false)},
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
		JWT:  JWTConfig{AccessSecret: env("JWT_ACCESS_SECRET", ""), RefreshSecret: env("JWT_REFRESH_SECRET", ""), AccessTTLMinutes: envInt("JWT_ACCESS_TTL_MINUTES", 15), RefreshTTLHours: envInt("JWT_REFRESH_TTL_HOURS", 168)},
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

package redis

import (
	"context"
	"fmt"

	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

type Client struct{ Client *goredis.Client }

func Connect(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	cli := goredis.NewClient(&goredis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB, DialTimeout: cfg.DialTimeout, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout})
	r := &Client{Client: cli}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.HealthTimeout)
	defer cancel()
	if err := r.Ping(pingCtx); err != nil {
		_ = cli.Close()
		return nil, err
	}
	return r, nil
}
func (r *Client) Ping(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}
func (r *Client) Close() error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Close()
}

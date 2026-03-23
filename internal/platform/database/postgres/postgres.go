package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

type Client struct {
	pool               *pgxpool.Pool
	healthCheckTimeout time.Duration
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	if cfg.SupabaseDBURL == "" {
		return nil, nil
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.SupabaseDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse supabase connection string: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.SupabaseDBMaxConns)
	poolConfig.MinConns = int32(cfg.SupabaseDBMinConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create supabase connection pool: %w", err)
	}

	client := &Client{
		pool:               pool,
		healthCheckTimeout: time.Duration(cfg.SupabaseDBHealthTimeoutSec) * time.Second,
	}

	if err := client.CheckHealth(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to supabase: %w", err)
	}

	return client, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.pool != nil
}

func (c *Client) CheckHealth(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}

	pingCtx := ctx
	cancel := func() {}
	if c.healthCheckTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, c.healthCheckTimeout)
	}
	defer cancel()

	return c.pool.Ping(pingCtx)
}

func (c *Client) Close() {
	if c.Enabled() {
		c.pool.Close()
	}
}

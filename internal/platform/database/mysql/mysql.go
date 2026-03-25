package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/truongcongminh96/ai-game-review-analyzer/config"
)

type Client struct {
	db                 *sql.DB
	healthCheckTimeout time.Duration
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	if cfg.DatabaseURL == "" {
		return nil, nil
	}

	db, err := sql.Open(config.DatabaseDriverMySQL, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	if cfg.DatabaseMaxConns > 0 {
		db.SetMaxOpenConns(cfg.DatabaseMaxConns)
	}

	maxIdleConns := cfg.DatabaseMinConns
	if cfg.DatabaseMaxConns > 0 && maxIdleConns > cfg.DatabaseMaxConns {
		maxIdleConns = cfg.DatabaseMaxConns
	}
	if maxIdleConns >= 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}

	client := &Client{
		db:                 db,
		healthCheckTimeout: time.Duration(cfg.DatabaseHealthTimeoutSec) * time.Second,
	}

	if err := client.CheckHealth(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}

	return client, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.db != nil
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

	return c.db.PingContext(pingCtx)
}

func (c *Client) Close() {
	if c.Enabled() {
		_ = c.db.Close()
	}
}

func (c *Client) DB() *sql.DB {
	if !c.Enabled() {
		return nil
	}

	return c.db
}

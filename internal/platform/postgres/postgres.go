package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string, attempts int, delay time.Duration) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	var pool *pgxpool.Pool
	for i := 0; i < attempts; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			pingErr := pool.Ping(ctx)
			if pingErr == nil {
				return pool, nil
			}
			err = pingErr
			pool.Close()
		}
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("connect postgres after %d attempts: %w", attempts, err)
}

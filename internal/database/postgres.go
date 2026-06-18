package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	"github.com/apekshita/eventpulse/internal/retry"
)

func OpenPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := retry.Do(ctx, 10, 500*time.Millisecond, func() error {
		return db.PingContext(ctx)
	}); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

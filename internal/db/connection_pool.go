package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPool is a wrapper over *pgxpool.Pool
type DBPool struct {
	pool *pgxpool.Pool
}

// InitDBPool initializes the DBPool by given *pgxpool.Config
func InitDBPool(ctx context.Context, config *pgxpool.Config) (DBPool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return DBPool{}, err
	}
	return DBPool{pool: pool}, nil
}

func (d *DBPool) Dispose() {
	d.pool.Close()
}

func (d *DBPool) Pool() *pgxpool.Pool {
	return d.pool
}

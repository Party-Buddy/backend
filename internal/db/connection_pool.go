package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBPool struct {
	pool *pgxpool.Pool
}

func InitDBPool() (DBPool, error) {
	pool, err := pgxpool.New(context.Background(), "")
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

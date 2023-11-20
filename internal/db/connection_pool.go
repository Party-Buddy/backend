package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPool is a wrapper over *pgxpool.Pool
type DBPool struct {
	pool *pgxpool.Pool
	ctx  context.Context
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

// AcquireTx acquires a connection from the pool, begins a new transaction, and provides it to f.
//
// The transaction is automatically rolled back after the function returns unless it commits the transaction.
func (d *DBPool) AcquireTx(ctx context.Context, f func(tx pgx.Tx) error) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	return f(tx)
}

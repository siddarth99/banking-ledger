package pkg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBConnection interface defines the methods needed for database operations
type PgDBConnection interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Close(ctx context.Context) error
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Transaction interface defines the methods needed for transaction operations
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Row interface defines the methods needed for row operations
type Row interface {
	Scan(dest ...any) error
}

// PostgresConn implements database operations using PostgreSQL
type PostgresConn struct {
	conn PgDBConnection
}

// NewPostgresProcessor creates a new PostgresProcessor with the given connection
func NewPostgresProcessor(conn PgDBConnection) *PostgresConn {
	return &PostgresConn{
		conn: conn,
	}
}

// BeginTx starts a new transaction
func (p *PostgresConn) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// Close closes the database connection
func (p *PostgresConn) Close(ctx context.Context) error {
	return p.conn.Close(ctx)
}

// ExecuteInTransaction executes the given function within a transaction
func (p *PostgresConn) ExecuteInTransaction(ctx context.Context, fn func(Transaction) error) error {
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Exec executes a query without returning any rows
func (p *PostgresConn) Exec(ctx context.Context, query string, args ...any) error {
	_, err := p.conn.Exec(ctx, query, args...)
	return err
}

// ExecWithTx executes a query within a transaction without returning any rows
func (p *PostgresConn) ExecWithTx(ctx context.Context, tx Transaction, query string, args ...any) error {
	_, err := tx.Exec(ctx, query, args...)
	return err
}

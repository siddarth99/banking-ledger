package pkg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockPgDBConnection implements the pkg.PgDBConnection interface for testing
type MockPgDBConnection struct {
	ExecFunc     func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	BeginFunc    func(ctx context.Context) (pgx.Tx, error)
	RollbackFunc func(ctx context.Context) error
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	CloseFunc    func(ctx context.Context) error
	CommitFunc   func(ctx context.Context) error
	ConnFunc     func() *pgx.Conn
	CopyFromFunc func(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowProvider pgx.CopyFromSource) (int64, error)
	QueryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	SendBatchFunc func(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	LargeObjectsFunc func() pgx.LargeObjects
	PrepareFunc func(ctx context.Context, name string, sql string) (*pgconn.StatementDescription, error)
}

// LargeObjects implements pgx.Tx.
func (m *MockPgDBConnection) LargeObjects() pgx.LargeObjects {
	if m.LargeObjectsFunc != nil {
		return m.LargeObjectsFunc()
	}
	return pgx.LargeObjects{}
}

// Prepare implements pgx.Tx.
func (m *MockPgDBConnection) Prepare(ctx context.Context, name string, sql string) (*pgconn.StatementDescription, error) {
	if m.PrepareFunc != nil {
		return m.PrepareFunc(ctx, name, sql)
	}
	return nil, nil
}

// Query implements pgx.Tx.
func (m *MockPgDBConnection) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return nil, nil
}

// SendBatch implements pgx.Tx.
func (m *MockPgDBConnection) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if m.SendBatchFunc != nil {
		return m.SendBatchFunc(ctx, b)
	}
	return nil
}

func (m *MockPgDBConnection) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowProvider pgx.CopyFromSource) (int64, error) {
	if m.CopyFromFunc != nil {
		return m.CopyFromFunc(ctx, tableName, columnNames, rowProvider)
	}
	return 0, nil
}

func (m *MockPgDBConnection) Conn() *pgx.Conn {
	if m.ConnFunc != nil {
		return m.ConnFunc()
	}
	return nil
}

func (m *MockPgDBConnection) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockPgDBConnection) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFunc != nil {
		return m.BeginFunc(ctx)
	}
	return nil, nil
}

func (m *MockPgDBConnection) Commit(ctx context.Context) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx)
	}
	return nil
}

func (m *MockPgDBConnection) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

func (m *MockPgDBConnection) Rollback(ctx context.Context) error {
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx)
	}
	return nil
}

func (m *MockPgDBConnection) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return nil
}

// MockPgxRow mocks pgx.Row
type MockPgxRow struct {
	ScanFunc func(dest ...interface{}) error
}

func (m *MockPgxRow) Scan(dest ...interface{}) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

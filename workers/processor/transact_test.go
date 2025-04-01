package processor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"github.com/siddarth99/banking-ledger/workers/processor"
	internal "github.com/siddarth99/banking-ledger/pkg"
)


func TestProcessTransaction_DepositSuccess(t *testing.T) {
	// Arrange
	capturedSQL := ""
	capturedArgs := []interface{}{}
	capturedIndex := ""
	capturedBody := ""

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					// Mock scanning balance and branch code
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 1000.00
					*branchCode = "BR001"
					return nil
				},
			}
		},
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = arguments
			return pgconn.CommandTag{}, nil
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	// Mock Elasticsearch client
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			capturedIndex = index
			bodyBytes, _ := io.ReadAll(body)
			capturedBody = string(bodyBytes)
			return &esapi.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
			}, nil
		},
	}

	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn:  mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        500.00,
			Type:          "DEPOSIT",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err)

	// Verify SQL update parameters
	assert.Contains(t, capturedSQL, "UPDATE accounts")
	assert.Equal(t, 1500.00, capturedArgs[0]) // 1000 (initial) + 500 (deposit)
	assert.Equal(t, "ACC123456", capturedArgs[1])

	// Verify Elasticsearch indexing
	assert.Contains(t, capturedIndex, "bank-transactions-")

	// Verify document content
	var doc map[string]interface{}
	err = json.Unmarshal([]byte(capturedBody), &doc)
	assert.NoError(t, err)
	assert.Equal(t, "TX123456", doc["transaction_id"])
	assert.Equal(t, "ACC123456", doc["account_number"])
	assert.Equal(t, 500.00, doc["amount"])
	assert.Equal(t, "DEPOSIT", doc["type"])
	assert.Equal(t, "COMPLETED", doc["status"])
	assert.Equal(t, "BR001", doc["branch_code"])
	assert.Equal(t, 1500.00, doc["balance_after_transaction"])
}

func TestProcessTransaction_WithdrawalSuccess(t *testing.T) {
	// Arrange
	capturedSQL := ""
	capturedArgs := []interface{}{}

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					// Mock scanning balance and branch code
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 1000.00
					*branchCode = "BR001"
					return nil
				},
			}
		},
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = arguments
			return pgconn.CommandTag{}, nil
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	// Mock Elasticsearch client
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
			}, nil
		},
	}

	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn:  mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        300.00,
			Type:          "WITHDRAWAL",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err)

	// Verify SQL update parameters
	assert.Contains(t, capturedSQL, "UPDATE accounts")
	assert.Equal(t, 700.00, capturedArgs[0]) // 1000 (initial) - 300 (withdrawal)
	assert.Equal(t, "ACC123456", capturedArgs[1])
	assert.Equal(t, 700.00, proc.Data.AvailableBalance)
}

func TestProcessTransaction_InvalidType(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 1000.00
					*branchCode = "BR001"
					return nil
				},
			}
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        500.00,
			Type:          "INVALID_TYPE", // Invalid transaction type
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err) // ProcessTransaction swallows errors
	assert.Contains(t, capturedLogs.String(), "invalid transaction type")
}

func TestProcessTransaction_InsufficientFunds(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 100.00 // Lower balance than withdrawal amount
					*branchCode = "BR001"
					return nil
				},
			}
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
			}, nil
		},
	}
	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn: mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        500.00,
			Type:          "WITHDRAWAL",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err) // ProcessTransaction swallows errors
	assert.Contains(t, capturedLogs.String(), "insufficient funds")
}

func TestProcessTransaction_NegativeAmount(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 1000.00
					*branchCode = "BR001"
					return nil
				},
			}
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
			}, nil
		},
	}
	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn: mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        -100.00, // Negative amount
			Type:          "DEPOSIT",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err) // ProcessTransaction swallows errors
	assert.Contains(t, capturedLogs.String(), "deposit amount must be positive")
}

func TestProcessTransaction_DatabaseError(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Create mock db connection with error
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return nil, errors.New("database connection error")
		},
	}
	
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return &esapi.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
			}, nil
		},
	}

	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn: mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        500.00,
			Type:          "DEPOSIT",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err) // ProcessTransaction swallows errors
	assert.Contains(t, capturedLogs.String(), "failed to begin transaction")
}

func TestProcessTransaction_ElasticsearchError(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Create mock tx
	mockTx := &internal.MockPgDBConnection{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &internal.MockPgxRow{
				ScanFunc: func(dest ...interface{}) error {
					balance := dest[0].(*float64)
					branchCode := dest[1].(*string)
					*balance = 1000.00
					*branchCode = "BR001"
					return nil
				},
			}
		},
	}

	// Create mock db connection
	mockConn := &internal.MockPgDBConnection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTx, nil
		},
	}

	// Mock Elasticsearch client with error
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return nil, errors.New("elasticsearch connection error")
		},
	}

	// Create processor with mocks
	proc := processor.TransactionProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn:  mockEsClient,
		},
		Data: processor.TransactionData{
			AccountNumber: "ACC123456",
			Amount:        500.00,
			Type:          "DEPOSIT",
			TransactionID: "TX123456",
		},
	}

	// Act
	err := proc.ProcessTransaction(context.Background())

	// Assert
	assert.NoError(t, err) // Should not fail the overall function
	assert.Contains(t, capturedLogs.String(), "Failed to index transaction in Elasticsearch")
}

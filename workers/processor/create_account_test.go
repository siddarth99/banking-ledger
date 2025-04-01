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
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"github.com/siddarth99/banking-ledger/workers/processor"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

// TransactionDocument represents a transaction document for testing
type TransactionDocument struct {
	TransactionID           string    `json:"transaction_id"`
	AccountNumber           string    `json:"account_number"`
	Amount                  float64   `json:"amount"`
	Type                    string    `json:"type"`
	Status                  string    `json:"status"`
	Timestamp               time.Time `json:"timestamp"`
	BranchCode              string    `json:"branch_code"`
	BalanceAfterTransaction float64   `json:"balance_after_transaction"`
}

func TestCreateAccount_Success(t *testing.T) {
	// Arrange
	capturedSQL := ""
	capturedArgs := []interface{}{}
	capturedIndex := ""
	capturedBody := ""

	// Mock PgxConn
	mockConn := &internal.MockPgDBConnection{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = arguments
			return pgconn.CommandTag{}, nil
		},
	}

	// Mock Elasticsearch client
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			capturedIndex = index
			bodyBytes, _ := io.ReadAll(body)
			capturedBody = string(bodyBytes)
			return internal.MockResponse(201, `{"result":"created"}`), nil
		},
	}

	// Create processor with mocks
	proc := processor.CreateAccountProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn:  mockEsClient,
		},
		Data: processor.AccountData{
			AccountHolderName: "Test User",
			InitialDeposit:    1000.00,
			BranchCode:        "BR1",
			ReferenceID:       "REF123456",
		},
	}

	// Act
	err := proc.CreateAccount(context.Background())

	// Assert
	assert.NoError(t, err)

	// Check that account number was generated
	assert.NotEmpty(t, proc.Data.AccountNumber)
	assert.True(t, strings.HasPrefix(proc.Data.AccountNumber, "BR1"))

	// Verify SQL query and parameters were correct
	assert.Contains(t, capturedSQL, "INSERT INTO accounts")
	assert.Equal(t, proc.Data.AccountNumber, capturedArgs[0])
	assert.Equal(t, "Test User", capturedArgs[1])
	assert.Equal(t, 1000.00, capturedArgs[2])
	assert.Equal(t, "BR1", capturedArgs[3])
	assert.Equal(t, "ACTIVE", capturedArgs[4])

	// Verify Elasticsearch indexing
	assert.Contains(t, capturedIndex, "bank-transactions-")

	// Verify document content
	var doc TransactionDocument
	err = json.Unmarshal([]byte(capturedBody), &doc)
	assert.NoError(t, err)
	assert.Equal(t, "REF123456", doc.TransactionID)
	assert.Equal(t, proc.Data.AccountNumber, doc.AccountNumber)
	assert.Equal(t, 1000.00, doc.Amount)
	assert.Equal(t, "DEPOSIT", doc.Type)
	assert.Equal(t, "COMPLETED", doc.Status)
	assert.Equal(t, "BR1", doc.BranchCode)
	assert.Equal(t, 1000.00, doc.BalanceAfterTransaction)
}

func TestCreateAccount_NegativeDeposit(t *testing.T) {
	// Arrange
	proc := processor.CreateAccountProcessor{
		Data: processor.AccountData{
			AccountHolderName: "Test User",
			InitialDeposit:    -100.00, // Negative deposit
			BranchCode:        "BR1",
			ReferenceID:       "REF123456",
		},
	}

	// Act
	err := proc.CreateAccount(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial Deposit cannot be negative")
}

func TestCreateAccount_DatabaseError(t *testing.T) {
	// Arrange
	expectedError := errors.New("database connection error")

	// Mock PgxConn
	mockConn := &internal.MockPgDBConnection{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, expectedError
		},
	}

	proc := processor.CreateAccountProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
		},
		Data: processor.AccountData{
			AccountHolderName: "Test User",
			InitialDeposit:    1000.00,
			BranchCode:        "BR1",
			ReferenceID:       "REF123456",
		},
	}

	// Act
	err := proc.CreateAccount(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create account")
}

func TestCreateAccount_ElasticsearchError(t *testing.T) {
	// Arrange
	capturedLogs := &bytes.Buffer{}
	log.SetOutput(capturedLogs)
	defer func() {
		log.SetOutput(nil) // Reset logger output
	}()

	// Mock PgxConn
	mockConn := &internal.MockPgDBConnection{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	}

	// Mock Elasticsearch client with error
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return nil, errors.New("elasticsearch connection error")
		},
	}

	proc := processor.CreateAccountProcessor{
		ProcessWorker: processor.ProcessWorker{
			PgxConn: mockConn,
			EsConn:  mockEsClient,
		},
		Data: processor.AccountData{
			AccountHolderName: "Test User",
			InitialDeposit:    1000.00,
			BranchCode:        "BR1",
			ReferenceID:       "REF123456",
		},
	}

	// Act
	err := proc.CreateAccount(context.Background())

	// Assert
	assert.NoError(t, err) // Should not fail the overall function
	assert.Contains(t, capturedLogs.String(), "Failed to index transaction in Elasticsearch")
	assert.Contains(t, capturedLogs.String(), "elasticsearch connection error")
}

func TestCreateAccount_AccountNumberGeneration(t *testing.T) {
	// Arrange
	accounts := make(map[string]bool)
	numAccounts := 100

	// Mock PgxConn that always succeeds
	mockConn := &internal.MockPgDBConnection{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	}

	// Mock Elasticsearch client that always succeeds
	mockEsClient := &internal.MockElasticsearchClient{
		IndexFunc: func(index string, body io.Reader) (*esapi.Response, error) {
			return internal.MockResponse(201, `{"result":"created"}`), nil
		},
	}

	// Act - Create multiple accounts and check for duplicate account numbers
	for i := 0; i < numAccounts; i++ {
		proc := processor.CreateAccountProcessor{
			ProcessWorker: processor.ProcessWorker{
				PgxConn: mockConn,
				EsConn:  mockEsClient,
			},
			Data: processor.AccountData{
				AccountHolderName: "Test User",
				InitialDeposit:    1000.00,
				BranchCode:        "BR001",
				ReferenceID:       "REF123456",
			},
		}

		err := proc.CreateAccount(context.Background())
		assert.NoError(t, err)

		// Store account number and check for duplicates
		accountNum := proc.Data.AccountNumber
		assert.False(t, accounts[accountNum], "Account number should be unique")
		accounts[accountNum] = true

		// Check that account number follows expected format
		assert.True(t, strings.HasPrefix(accountNum, "BR001"))
		assert.Len(t, accountNum, len("BR001")+7) // Branch code + 7 digits
	}

	// Assert - All account numbers should be unique
	assert.Equal(t, numAccounts, len(accounts), "All account numbers should be unique")
}

package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type TransactionProcessor struct {
	ProcessWorker
	Data TransactionData
}

// TransactionData represents the data needed for a transaction
type TransactionData struct {
	AccountNumber     string  `json:"accountNumber"`
	Amount            float64 `json:"amount"`
	AvailableBalance  float64 `json:"availableBalance"`
	Type              string  `json:"type"` // "DEPOSIT" or "WITHDRAWAL"
	TransactionID     string  `json:"transactionId"`
	BranchCode        string  `json:"branchCode"`
}

func (p *TransactionProcessor) transact(ctx context.Context) error {
	// Start a transaction
	tx, err := p.PgxConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Will be ignored if transaction is committed

	// Get current account balance
	var currentBalance float64
	var branchCode string
	query := `SELECT available_balance, branch_code FROM accounts 
	WHERE account_number = $1 AND status='ACTIVE' FOR UPDATE;`

	err = tx.QueryRow(ctx, query, p.Data.AccountNumber).Scan(&currentBalance, &branchCode)
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	p.Data.BranchCode = branchCode
	p.Data.AvailableBalance = currentBalance

	// Calculate new balance based on transaction type
	var newBalance float64
	switch p.Data.Type {
	case "DEPOSIT":
		if p.Data.Amount <= 0 {
			return fmt.Errorf("deposit amount must be positive")
		}
		newBalance = currentBalance + p.Data.Amount
	case "WITHDRAWAL":
		if p.Data.Amount <= 0 {
			return fmt.Errorf("withdrawal amount must be positive")
		}
		if currentBalance < p.Data.Amount {
			return fmt.Errorf("insufficient funds")
		}
		newBalance = currentBalance - p.Data.Amount
	default:
		return fmt.Errorf("invalid transaction type: %s", p.Data.Type)
	}

	p.Data.AvailableBalance = newBalance

	// Update account balance
	updateQuery := `UPDATE accounts SET available_balance = $1 WHERE account_number = $2`
	_, err = tx.Exec(ctx, updateQuery, newBalance, p.Data.AccountNumber)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ProcessTransaction handles deposit and withdrawal transactions
func (p *TransactionProcessor) ProcessTransaction(ctx context.Context) error {

	err := p.transact(ctx)
	status := "COMPLETED"
	if err != nil {
		log.Println(err)
		status = "FAILED"
	}
	transactionDoc := TransactionDocument{
		AccountNumber: p.Data.AccountNumber,
		Type:          p.Data.Type,
		Amount:        p.Data.Amount,
		TransactionID: p.Data.TransactionID,
		Timestamp:     time.Now(),
		BranchCode:    p.Data.BranchCode,
		Status:        status,
		BalanceAfterTransaction: p.Data.AvailableBalance,
	}

	// Create index name with date format for better data management
	indexName := fmt.Sprintf("bank-transactions-%s", time.Now().Format("2006-01-02"))

	// Index the transaction document
	transactionDocJSON, err := json.Marshal(transactionDoc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}

	req := strings.NewReader(string(transactionDocJSON))
	res, err := p.EsConn.Index(indexName, req)
	if err != nil {
		// Log the error but don't fail the account creation
		log.Printf("Failed to index transaction in Elasticsearch: %v", err)
	}

	if res != nil {
		defer res.Body.Close()
	}

	return nil
}

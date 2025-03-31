package processor

import (
	"time"

	"github.com/jackc/pgx/v5"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

type ProcessWorker struct {
	PgxConn *pgx.Conn
	EsConn  internal.ElasticsearchClient
}

// Log the account creation transaction to Elasticsearch
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
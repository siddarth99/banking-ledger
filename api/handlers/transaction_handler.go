package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

// TransactionRequest represents the data structure for transaction requests
// It contains all necessary fields required to process a transaction
type TransactionRequest struct {
	AccountNumber   string  `json:"accountNumber" binding:"required"` // Account number for the transaction
	Amount          float64 `json:"amount" binding:"required"`        // Amount to debit or credit
	TransactionType string  `json:"type" binding:"required"`          // Type of transaction (debit/credit)
	Description     string  `json:"description"`                      // Optional description
	TransactionID   string  `json:"transactionId"`                    // Unique identifier for tracking
}

// transactionResponse represents the response structure sent back to clients
// after a successful transaction request
type transactionResponse struct {
	TransactionID string    `json:"transactionID"` // Unique ID for the transaction
	CreatedAt     time.Time `json:"createdAt"`     // Timestamp when the transaction request was processed
}

// TransactionHandler creates a new HTTP handler for transaction requests
// It takes a context, an AMQP channel, and a queue name for message publishing
// Returns a gin.HandlerFunc that can be registered with the router
func TransactionHandler(ctx context.Context, amqpChannel internal.AMQPChannel, queueName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var transactionRequestJson TransactionRequest
		// Parse and validate the incoming JSON request
		if err := c.ShouldBindJSON(&transactionRequestJson); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"errorCode": http.StatusBadRequest,
				"error":     err.Error(),
			})
			return
		}

		// Process the transaction request
		response, err := transactionRequestJson.createTransaction(ctx, amqpChannel, queueName)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     err.Error(),
			})
			return
		}

		// Return a successful response with tracking information
		c.JSON(http.StatusAccepted, gin.H{
			"transactionID": response.TransactionID,
			"createdAt":     response.CreatedAt,
		})
	}
}

// createTransaction processes a transaction request by:
// 1. Generating unique reference and transaction IDs
// 2. Publishing the request to a message queue for asynchronous processing
// 3. Returning a response with tracking information
func (t *TransactionRequest) createTransaction(ctx context.Context, amqpChannel internal.AMQPChannel, queueName string) (transactionResponse, error) {

	// Generate a unique transaction ID
	transactionID := uuid.New().String()

	t.TransactionID = transactionID

	// Create JSON payload
	requestByteArray, err := json.Marshal(t)
	if err != nil {
		// Handle marshaling error
		fmt.Printf("Error while marshalling transaction request: %s", err.Error())
		return transactionResponse{}, err
	}

	// Publish message to RabbitMQ for asynchronous processing
	err = internal.PublishWithContext(
		ctx,
		requestByteArray,
		amqpChannel,
		"",        // default exchange
		queueName, // routing key = queue name
		false,     // mandatory
		false,     // immediate
	)

	if err != nil {
		// Handle publishing error
		fmt.Printf("Error while publishing transaction request to queue: %s", err.Error())
		return transactionResponse{}, err
	}

	// Return response with tracking information
	return transactionResponse{
		TransactionID: transactionID,
		CreatedAt:     time.Now(),
	}, nil
}

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

// AccountRequest represents the data structure for account creation requests
// It contains all necessary fields required to create a new bank account
type AccountRequest struct {
	AccountHolderName string  `json:"accountHolderName" binding:"required"` // Name of the account holder
	BranchCode        string  `json:"branchCode" binding:"required"`        // 3-character branch code
	InitialDeposit    float64 `json:"initialDeposit" binding:"required"`    // Initial amount to deposit
	ReferenceID       string  `json:"referenceID"`                          // Unique identifier for tracking
}

// accountResponse represents the response structure sent back to clients
// after a successful account creation request
type accountResponse struct {
	ReferenceID string    `json:"referenceID"` // Unique identifier for tracking the request
	CreatedAt   time.Time `json:"createdAt"`   // Timestamp when the account creation request was processed
}

// CreateAccountHandler creates a new HTTP handler for account creation requests
// It takes a context, an AMQP channel, and a queue name for message publishing
// Returns a gin.HandlerFunc that can be registered with the router
func CreateAccountHandler(ctx context.Context, amqpChannel internal.AMQPChannel, queueName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var accountRequestJson AccountRequest
		// Parse and validate the incoming JSON request
		if err := c.ShouldBindJSON(&accountRequestJson); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"errorCode": http.StatusBadRequest,
				"error":     err.Error(),
			})
			return
		}

		// Process the account creation request
		response, err := accountRequestJson.createAccount(ctx, amqpChannel, queueName)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     err.Error(),
			})
			return
		}

		// Return a successful response with tracking information
		c.JSON(http.StatusAccepted, gin.H{
			"referenceID": response.ReferenceID,
			"createdAt":   response.CreatedAt,
		})
	}
}

// createAccount processes an account creation request by:
// 1. Generating a unique reference ID
// 2. Publishing the request to a message queue for asynchronous processing
// 3. Returning a response with tracking information
func (a *AccountRequest) createAccount(ctx context.Context, amqpChannel internal.AMQPChannel, queueName string) (accountResponse, error) {
	// Generate a unique reference ID for tracking this request
	a.ReferenceID = uuid.New().String()

	// Create JSON payload
	requestByteArray, err := json.Marshal(a)
	if err != nil {
		// Handle marshaling error
		fmt.Printf("Error while Marshalling account request %s", err.Error())
		return accountResponse{}, err
	}

	// Publish message to RabbitMQ for asynchronous processing
	err = internal.PublishWithContext(
		ctx,
		requestByteArray,
		amqpChannel,
		"",        // default exchange
		queueName, // routing key = queue name
		false,     // mandatory - don't require the message to be routed to a queue
		false,     // immediate - don't require immediate delivery to a consumer
	)

	if err != nil {
		// Handle publishing error
		fmt.Printf("Error while Publishing account request to queue %s", err.Error())
		return accountResponse{}, err
	}

	// Return response with tracking ID and timestamp
	return accountResponse{
		ReferenceID: a.ReferenceID,
		CreatedAt:   time.Now(),
	}, nil
}

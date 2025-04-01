package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

// Test for TransactionHandler
func TestTransactionHandler(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("Valid request should return 200", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "transaction_queue", false, false, mock.Anything).Return(nil)

		// Create a test request
		validRequest := TransactionRequest{
			AccountNumber:   "ACC123456",
			Amount:          100.50,
			TransactionType: "credit",
			Description:     "Test transaction",
		}
		jsonValue, _ := json.Marshal(validRequest)
		req, _ := http.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/transactions", TransactionHandler(context.Background(), mockChannel, "transaction_queue"))
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response contains expected fields
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "transactionID")
		assert.Contains(t, response, "createdAt")

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})

	t.Run("Invalid request should return 400", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)

		// Create an invalid request (missing required fields)
		invalidRequest := TransactionRequest{
			// Missing AccountNumber and Amount
			TransactionType: "debit",
		}
		jsonValue, _ := json.Marshal(invalidRequest)
		req, _ := http.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/transactions", TransactionHandler(context.Background(), mockChannel, "transaction_queue"))
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify our mock was never called
		mockChannel.AssertNotCalled(t, "PublishWithContext")
	})

	t.Run("Publishing error should return 500", func(t *testing.T) {
		// Setup mock channel that returns an error
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "transaction_queue", false, false, mock.Anything).Return(errors.New("publish error"))

		// Create a valid request
		validRequest := TransactionRequest{
			AccountNumber:   "ACC123456",
			Amount:          100.50,
			TransactionType: "debit",
			Description:     "Test transaction",
		}
		jsonValue, _ := json.Marshal(validRequest)
		req, _ := http.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/transactions", TransactionHandler(context.Background(), mockChannel, "transaction_queue"))
		router.ServeHTTP(w, req)

		// Assert response (note: transaction handler returns 500 for publishing errors, not 400)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})
}

// Test for createTransaction method
func TestCreateTransaction(t *testing.T) {
	t.Run("Successful transaction creation", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "test_queue", false, false, mock.Anything).Return(nil)

		// Create request
		req := &TransactionRequest{
			AccountNumber:   "ACC123456",
			Amount:          200.75,
			TransactionType: "credit",
			Description:     "Test transaction",
		}

		// Call method
		resp, err := req.createTransaction(context.Background(), mockChannel, "test_queue")

		// Assert no error
		assert.NoError(t, err)

		// Verify response
		assert.NotEmpty(t, resp.TransactionID)
		assert.NotZero(t, resp.CreatedAt)
		assert.WithinDuration(t, time.Now(), resp.CreatedAt, 2*time.Second)

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})

	t.Run("Publishing error", func(t *testing.T) {
		// Setup mock channel that returns an error
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "test_queue", false, false, mock.Anything).Return(errors.New("publish error"))

		// Create request
		req := &TransactionRequest{
			AccountNumber:   "ACC123456",
			Amount:          200.75,
			TransactionType: "debit",
			Description:     "Test transaction",
		}

		// Call method
		resp, err := req.createTransaction(context.Background(), mockChannel, "test_queue")

		// Assert error
		assert.Error(t, err)
		assert.Equal(t, "publish error", err.Error())

		// Verify response is empty
		assert.Empty(t, resp.TransactionID)
		assert.True(t, resp.CreatedAt.IsZero())

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})
}

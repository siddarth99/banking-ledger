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

// Test for CreateAccountHandler
func TestCreateAccountHandler(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("Valid request should return 200", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "account_queue", false, false, mock.Anything).Return(nil)

		// Create a test request
		validRequest := AccountRequest{
			AccountHolderName: "John Doe",
			BranchCode:        "ABC",
			InitialDeposit:    1000.00,
		}
		jsonValue, _ := json.Marshal(validRequest)
		req, _ := http.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/accounts", CreateAccountHandler(context.Background(), mockChannel, "account_queue"))
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response contains expected fields
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "referenceID")
		assert.Contains(t, response, "createdAt")

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})

	t.Run("Invalid request should return 400", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)

		// Create an invalid request (missing required fields)
		invalidRequest := AccountRequest{
			// Missing AccountHolderName and BranchCode
			InitialDeposit: 1000.00,
		}
		jsonValue, _ := json.Marshal(invalidRequest)
		req, _ := http.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/accounts", CreateAccountHandler(context.Background(), mockChannel, "account_queue"))
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
			mock.Anything, "", "account_queue", false, false, mock.Anything).Return(errors.New("publish error"))

		// Create a valid request
		validRequest := AccountRequest{
			AccountHolderName: "John Doe",
			BranchCode:        "ABC",
			InitialDeposit:    1000.00,
		}
		jsonValue, _ := json.Marshal(validRequest)
		req, _ := http.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Setup response recorder
		w := httptest.NewRecorder()

		// Setup router
		router := gin.Default()
		router.POST("/accounts", CreateAccountHandler(context.Background(), mockChannel, "account_queue"))
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})
}

// Test for createAccount method
func TestCreateAccount(t *testing.T) {
	t.Run("Successful account creation", func(t *testing.T) {
		// Setup mock channel
		mockChannel := new(internal.MockAMQPChannel)
		mockChannel.On("PublishWithContext",
			mock.Anything, "", "test_queue", false, false, mock.Anything).Return(nil)

		// Create request
		req := &AccountRequest{
			AccountHolderName: "Jane Doe",
			BranchCode:        "XYZ",
			InitialDeposit:    500.00,
		}

		// Call method
		resp, err := req.createAccount(context.Background(), mockChannel, "test_queue")

		// Assert no error
		assert.NoError(t, err)

		// Verify response
		assert.NotEmpty(t, resp.ReferenceID)
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
		req := &AccountRequest{
			AccountHolderName: "Jane Doe",
			BranchCode:        "XYZ",
			InitialDeposit:    500.00,
		}

		// Call method
		resp, err := req.createAccount(context.Background(), mockChannel, "test_queue")

		// Assert error
		assert.Error(t, err)
		assert.Equal(t, "publish error", err.Error())

		// Verify response is empty
		assert.Empty(t, resp.ReferenceID)
		assert.True(t, resp.CreatedAt.IsZero())

		// Verify our mock was called as expected
		mockChannel.AssertExpectations(t)
	})
}

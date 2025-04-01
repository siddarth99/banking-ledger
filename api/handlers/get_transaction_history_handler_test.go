package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockElasticsearchClient implements the ElasticsearchClient interface for testing
type MockElasticsearchClient struct {
	mock.Mock
}

func (m *MockElasticsearchClient) Search(indices []string, body io.Reader) (*esapi.Response, error) {
	args := m.Called(indices, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*esapi.Response), args.Error(1)
}

func (m *MockElasticsearchClient) Index(index string, body io.Reader) (*esapi.Response, error) {
	args := m.Called(index, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*esapi.Response), args.Error(1)
}

func (m *MockElasticsearchClient) Info() (*esapi.Response, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*esapi.Response), args.Error(1)
}

// TestGetTransactionHistoryHandler tests the GetTransactionHistoryHandler function
func TestGetTransactionHistoryHandler(t *testing.T) {
	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)

	// Test case: Successful retrieval of transaction history
	t.Run("Success response with transactions", func(t *testing.T) {
		// Create mock client
		mockClient := new(MockElasticsearchClient)

		// Create sample Elasticsearch response
		timestamp := time.Now()
		responseBody := map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 2,
				},
				"hits": []map[string]interface{}{
					{
						"_source": map[string]interface{}{
							"transaction_id":            "tx123",
							"amount":                    100.50,
							"type":                      "credit",
							"status":                    "completed",
							"timestamp":                 timestamp,
							"balance_after_transaction": 500.75,
							"description":               "Salary deposit",
						},
					},
					{
						"_source": map[string]interface{}{
							"transaction_id":            "tx456",
							"amount":                    50.25,
							"type":                      "debit",
							"status":                    "completed",
							"timestamp":                 timestamp.Add(-24 * time.Hour),
							"balance_after_transaction": 400.25,
							"description":               "ATM withdrawal",
						},
					},
				},
			},
		}

		responseJSON, _ := json.Marshal(responseBody)
		mockResponse := &esapi.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
		}

		// Setup mock expectations
		mockClient.On("Search",
			[]string{"bank-transactions-*"},
			mock.MatchedBy(func(buf io.Reader) bool {
				// Verify query contains account number
				var query map[string]interface{}
				if err := json.NewDecoder(buf).Decode(&query); err != nil {
					return false
				}
				match, ok := query["query"].(map[string]interface{})["match"].(map[string]interface{})
				return ok && match["account_number"] == "ACC123"
			})).Return(mockResponse, nil)

		// Setup router
		router := gin.New()
		router.GET("/account/:accountNumber/history", GetTransactionHistoryHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/ACC123/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)


		var response TransactionHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		
		fmt.Println(response)

		// Verify response data
		assert.Equal(t, "ACC123", response.AccountNumber)
		assert.Equal(t, 2, response.TotalCount)
		assert.Equal(t, 1, response.CurrentPage)
		assert.Len(t, response.Transactions, 2)

		// Verify first transaction details
		assert.Equal(t, "tx123", response.Transactions[0].TransactionID)
		assert.Equal(t, 100.50, response.Transactions[0].Amount)
		assert.Equal(t, "credit", response.Transactions[0].TransactionType)
		assert.Equal(t, "completed", response.Transactions[0].Status)
		assert.Equal(t, 500.75, response.Transactions[0].BalanceAfterTransaction)
		assert.Equal(t, "Salary deposit", response.Transactions[0].Description)

		// Verify second transaction details
		assert.Equal(t, "tx456", response.Transactions[1].TransactionID)
		assert.Equal(t, "debit", response.Transactions[1].TransactionType)

		// Verify our expectations were met
		mockClient.AssertExpectations(t)
	})

	// Test case: Empty account parameter
	t.Run("Missing account number", func(t *testing.T) {
		mockClient := new(MockElasticsearchClient)

		router := gin.New()
		router.GET("/account/:accountNumber/history", GetTransactionHistoryHandler(mockClient))

		// Create request without account number
		req, _ := http.NewRequest("GET", "/account//history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusBadRequest), response["errorCode"])

		// Verify no Elasticsearch calls were made
		mockClient.AssertNotCalled(t, "Search")
	})

	// Test case: Elasticsearch error
	t.Run("Elasticsearch client error", func(t *testing.T) {
		mockClient := new(MockElasticsearchClient)

		// Setup mock to return error
		mockClient.On("Search", mock.Anything, mock.Anything).Return(nil, errors.New("connection refused"))

		router := gin.New()
		router.GET("/account/:accountNumber/history", GetTransactionHistoryHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/ACC123/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusInternalServerError), response["errorCode"])
		assert.True(t, strings.Contains(response["error"].(string), "connection refused"))

		// Verify expectations
		mockClient.AssertExpectations(t)
	})

	// Test case: Elasticsearch returns error response
	t.Run("Elasticsearch error response", func(t *testing.T) {
		mockClient := new(MockElasticsearchClient)

		// Create error response
		errorResponseBody := map[string]interface{}{
			"error": map[string]interface{}{
				"type":   "index_not_found_exception",
				"reason": "no such index [bank-transactions]",
			},
		}
		responseJSON, _ := json.Marshal(errorResponseBody)
		mockResponse := &esapi.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
		}

		// Setup mock to return error response
		mockClient.On("Search", mock.Anything, mock.Anything).Return(mockResponse, nil)

		router := gin.New()
		router.GET("/account/:accountNumber/history", GetTransactionHistoryHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/ACC123/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusInternalServerError), response["errorCode"])
		assert.True(t, strings.Contains(response["error"].(string), "index_not_found_exception"))

		// Verify expectations
		mockClient.AssertExpectations(t)
	})

	// Test case: Pagination parameter
	t.Run("Pagination works correctly", func(t *testing.T) {
		mockClient := new(MockElasticsearchClient)

		// Create empty response
		responseBody := map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 30, // Total of 30 items
				},
				"hits": []map[string]interface{}{}, // Empty list for this test
			},
		}
		responseJSON, _ := json.Marshal(responseBody)
		mockResponse := &esapi.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
		}

		// Setup mock to verify pagination
		mockClient.On("Search",
			[]string{"bank-transactions-*"},
			mock.MatchedBy(func(buf *bytes.Buffer) bool {
				// Verify pagination parameters
				var query map[string]interface{}
				if err := json.NewDecoder(buf).Decode(&query); err != nil {
					return false
				}
				return query["from"] == float64(20) && query["size"] == float64(10)
			})).Return(mockResponse, nil)

		router := gin.New()
		router.GET("/account/:accountNumber/history", GetTransactionHistoryHandler(mockClient))

		// Create request with page parameter
		req, _ := http.NewRequest("GET", "/account/ACC123/history?page=3", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response TransactionHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 3, response.CurrentPage) // Should reflect page 3

		// Verify expectations
		mockClient.AssertExpectations(t)
	})
}

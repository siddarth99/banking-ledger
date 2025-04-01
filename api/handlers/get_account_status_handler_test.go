package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gin-gonic/gin"
	"github.com/siddarth99/banking-ledger/api/handlers"
	internal "github.com/siddarth99/banking-ledger/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAccountStatusHandler(t *testing.T) {
	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)

	// Test case: Successful retrieval of account status (ACTIVE)
	t.Run("Success response with ACTIVE status", func(t *testing.T) {

		mockClient := new(internal.MockElasticsearchClient)

		// Create sample Elasticsearch response
		responseBody := map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 1,
				},
				"hits": []map[string]interface{}{
					{
						"_source": map[string]interface{}{
							"transaction_id": "ref-123",
							"status":         "COMPLETED",
							"account_number": "ACC123456",
							"message":        "Account created successfully",
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
			mock.MatchedBy(func(reader io.Reader) bool {
				// Verify query contains reference ID
				var query map[string]interface{}
				if err := json.NewDecoder(reader).Decode(&query); err != nil {
					return false
				}
				match, ok := query["query"].(map[string]interface{})["match"].(map[string]interface{})
				return ok && match["transaction_id"] == "ref-123"
			})).Return(mockResponse, nil)

		// Setup router
		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/status/ref-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.AccountStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify response data
		assert.Equal(t, "ref-123", response.ReferenceID)
		assert.Equal(t, "ACTIVE", response.Status)
		assert.Equal(t, "ACC123456", response.AccountNumber)

		// Verify our expectations were met
		mockClient.AssertExpectations(t)
	})

	// Test case: Successful retrieval of account status (INACTIVE)
	t.Run("Success response with INACTIVE status", func(t *testing.T) {
		// Create mock client
		mockClient := new(internal.MockElasticsearchClient)

		// Create sample Elasticsearch response
		responseBody := map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 1,
				},
				"hits": []map[string]interface{}{
					{
						"_source": map[string]interface{}{
							"transaction_id": "ref-456",
							"status":         "PENDING",
							"message":        "Account creation in progress",
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
			mock.MatchedBy(func(reader io.Reader) bool {
				var query map[string]interface{}
				if err := json.NewDecoder(reader).Decode(&query); err != nil {
					return false
				}
				match, ok := query["query"].(map[string]interface{})["match"].(map[string]interface{})
				return ok && match["transaction_id"] == "ref-456"
			})).Return(mockResponse, nil)

		// Setup router
		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/status/ref-456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response handlers.AccountStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify response data
		assert.Equal(t, "ref-456", response.ReferenceID)
		assert.Equal(t, "INACTIVE", response.Status)
		assert.Empty(t, response.AccountNumber)

		// Verify our expectations were met
		mockClient.AssertExpectations(t)
	})

	// Test case: Empty reference ID parameter
	t.Run("Missing reference ID", func(t *testing.T) {
		mockClient := new(internal.MockElasticsearchClient)

		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request without reference ID
		req, _ := http.NewRequest("GET", "/account/status/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusNotFound, w.Code) // Gin returns 404 for missing path parameters

		// Verify no Elasticsearch calls were made
		mockClient.AssertNotCalled(t, "Search")
	})

	// Test case: Elasticsearch error
	t.Run("Elasticsearch client error", func(t *testing.T) {
		mockClient := new(internal.MockElasticsearchClient)

		// Setup mock to return error
		mockClient.On("Search", mock.Anything, mock.Anything).Return(nil, errors.New("connection refused"))

		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/status/ref-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusInternalServerError), response["errorCode"])
		assert.Contains(t, response["error"].(string), "Failed to search account status")

		// Verify expectations
		mockClient.AssertExpectations(t)
	})

	// Test case: Reference ID not found
	t.Run("Reference ID not found", func(t *testing.T) {
		mockClient := new(internal.MockElasticsearchClient)

		// Create empty response
		responseBody := map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 0,
				},
				"hits": []map[string]interface{}{},
			},
		}
		responseJSON, _ := json.Marshal(responseBody)
		mockResponse := &esapi.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
		}

		// Setup mock expectations
		mockClient.On("Search", mock.Anything, mock.Anything).Return(mockResponse, nil)

		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/status/unknown-ref", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusNotFound), response["errorCode"])
		assert.Contains(t, response["error"].(string), "Account creation request not found")

		// Verify expectations
		mockClient.AssertExpectations(t)
	})

	// Test case: Elasticsearch error response
	t.Run("Elasticsearch error response", func(t *testing.T) {
		mockClient := new(internal.MockElasticsearchClient)

		// Create error response
		mockResponse := &esapi.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"index_not_found_exception"}`))),
		}

		// Setup mock to return error response
		mockClient.On("Search", mock.Anything, mock.Anything).Return(mockResponse, nil)

		router := gin.New()
		router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(mockClient))

		// Create request
		req, _ := http.NewRequest("GET", "/account/status/ref-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(http.StatusInternalServerError), response["errorCode"])
		assert.Contains(t, response["error"].(string), "Error response from Elasticsearch")

		// Verify expectations
		mockClient.AssertExpectations(t)
	})
}

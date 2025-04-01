package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

// AccountStatusResponse represents the response structure for account status
type AccountStatusResponse struct {
	ReferenceID   string `json:"referenceID"`
	Status        string `json:"status"`
	AccountNumber string `json:"accountNumber,omitempty"`
	Message       string `json:"message,omitempty"`
}

// GetAccountStatusHandler handles requests to check account creation status
func GetAccountStatusHandler(esClient internal.ElasticsearchClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		referenceID := c.Param("referenceId")
		if referenceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"errorCode": http.StatusBadRequest,
				"error":     "Reference ID is required",
			})
			return
		}

		// Construct Elasticsearch query
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"match": map[string]interface{}{
					"transaction_id": referenceID,
				},
			},
		}

		// Convert query to bytes
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     "Failed to construct search query",
			})
			return
		}

		// Search in Elasticsearch
		res, err := esClient.Search([]string{"bank-transactions-*"}, &buf)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     "Failed to search account status",
			})
			return
		}
		defer res.Body.Close()

		if res.IsError() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     "Error response from Elasticsearch",
			})
			return
		}

		// Parse the response
		var result map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errorCode": http.StatusInternalServerError,
				"error":     "Failed to parse search results",
			})
			return
		}

		// Check if any hits were found
		hits := result["hits"].(map[string]interface{})
		if hits["total"].(map[string]interface{})["value"].(float64) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"errorCode": http.StatusNotFound,
				"error":     "Account creation request not found",
			})
			return
		}

		// Get the first hit
		hit := hits["hits"].([]interface{})[0].(map[string]interface{})
		source := hit["_source"].(map[string]interface{})

		// Construct response
		response := AccountStatusResponse{
			ReferenceID: referenceID,
		}

		if source["status"] == "COMPLETED" {
			response.Status = "ACTIVE"
		} else {
			response.Status = "INACTIVE"
		}

		// Add account number if status is ACTIVE
		if response.Status == "ACTIVE" {
			response.AccountNumber = source["account_number"].(string)
		}

		// Add message if present
		if message, ok := source["message"].(string); ok {
			response.Message = message
		}

		c.JSON(http.StatusOK, response)
	}
}

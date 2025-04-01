package pkg

import (
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// MockElasticsearchClient provides a mock implementation for testing
type MockElasticsearchClient struct {
	IndexFunc  func(index string, body io.Reader) (*esapi.Response, error)
	SearchFunc func(indices []string, body io.Reader) (*esapi.Response, error)
	GetFunc    func(index, id string) (*esapi.Response, error)
	DeleteFunc func(index, id string) (*esapi.Response, error)
	BulkFunc   func(body io.Reader) (*esapi.Response, error)
	InfoFunc   func() (*esapi.Response, error)
}

// Index implements the mock Index method
func (m *MockElasticsearchClient) Index(index string, body io.Reader) (*esapi.Response, error) {
	if m.IndexFunc != nil {
		return m.IndexFunc(index, body)
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// Search implements the mock Search method
func (m *MockElasticsearchClient) Search(indices []string, body io.Reader) (*esapi.Response, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(indices, body)
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// Get implements the mock Get method
func (m *MockElasticsearchClient) Get(index, id string) (*esapi.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(index, id)
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// Delete implements the mock Delete method
func (m *MockElasticsearchClient) Delete(index, id string) (*esapi.Response, error) {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(index, id)
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// Bulk implements the mock Bulk method
func (m *MockElasticsearchClient) Bulk(body io.Reader) (*esapi.Response, error) {
	if m.BulkFunc != nil {
		return m.BulkFunc(body)
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// Info implements the mock Info method
func (m *MockElasticsearchClient) Info() (*esapi.Response, error) {
	if m.InfoFunc != nil {
		return m.InfoFunc()
	}
	return &esapi.Response{StatusCode: 200}, nil
}

// MockResponse creates a mock response for testing
func MockResponse(statusCode int, body string) *esapi.Response {
	return &esapi.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

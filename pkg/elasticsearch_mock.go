package pkg

import (
	"io"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/go-elasticsearch/v8/esapi"
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
package pkg

import (
	"context"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchClient defines an interface for Elasticsearch operations
type ElasticsearchClient interface {
	// Index indexes a document in Elasticsearch
	Index(index string, body io.Reader) (*esapi.Response, error)
	// Search searches the index in Elasticsearch
	Search(indices []string, body io.Reader) (*esapi.Response, error)
	// Info returns info about the cluster
	Info() (*esapi.Response, error)
}

// ElasticsearchWrapper wraps the elasticsearch client to implement the ElasticsearchClient interface
type ElasticsearchWrapper struct {
	client *elasticsearch.Client
}

// NewElasticsearchClient creates a new Elasticsearch client wrapper
func NewElasticsearchClient(config elasticsearch.Config) (ElasticsearchClient, error) {
	client, err := elasticsearch.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ElasticsearchWrapper{
		client: client,
	}, nil
}

// Index indexes a document in Elasticsearch
func (e *ElasticsearchWrapper) Index(index string, body io.Reader) (*esapi.Response, error) {
	return e.client.Index(index, body)
}

// Search performs a search query
func (e *ElasticsearchWrapper) Search(indices []string, body io.Reader) (*esapi.Response, error) {
	return e.client.Search(
		e.client.Search.WithContext(context.Background()),
		e.client.Search.WithIndex(indices...),
		e.client.Search.WithBody(body),
	)
}

// Get retrieves a document by ID
func (e *ElasticsearchWrapper) Get(index, id string) (*esapi.Response, error) {
	return e.client.Get(
		index,
		id,
	)
}

// Info returns info about the cluster
func (e *ElasticsearchWrapper) Info() (*esapi.Response, error) {
	return e.client.Info()
}
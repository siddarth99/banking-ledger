package pkg

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
)

// MockAMQPChannel is a mock implementation of the internal.AMQPChannel interface
type MockAMQPChannel struct {
	mock.Mock
}

func (m *MockAMQPChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAMQPChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

func (m *MockAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, table amqp.Table) (amqp.Queue, error) {
	args := m.Called(name, durable, autoDelete, exclusive, noWait, nil)
	return amqp.Queue{}, args.Error(0)
}

func (m *MockAMQPChannel) Consume(name,	consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	mockArgs := m.Called(name, consumer, autoAck, exclusive, noLocal, noWait, nil)
	return mockArgs.Get(0).(<-chan amqp.Delivery), mockArgs.Error(1)
}
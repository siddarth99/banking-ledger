package pkg

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPConnectionCloser defines an interface for closing AMQP connections
type AMQPConnectionCloser interface {
	Close() error
}

type AMQPConnection interface {
	AMQPConnectionCloser
	AMQPChannelCreator
}

// AMQPChannelCreator defines an interface for creating AMQP channels
type AMQPChannelCreator interface {
	Channel() (AMQPChannel, error)
}

// ChannelCloser defines an interface for closing AMQP channels
type AMQPChannelCloser interface {
	Close() error
}

type AMPQPQueuePublisher interface {
	PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error
}

// AMQPQueueDeclarer defines an interface for declaring AMQP queues
type AMQPQueueDeclarer interface {
	QueueDeclare(string, bool, bool, bool, bool, amqp.Table) (amqp.Queue, error)
}

// AMQPChannel combines the capabilities needed from a channel
type AMQPChannel interface {
	AMQPChannelCloser
	AMQPQueueDeclarer
	AMPQPQueuePublisher
}

// AMQPConnectionWrapper wraps an amqp.Connection to implement AMQPChannelCreator
type AMQPConnectionWrapper struct {
	conn *amqp.Connection
}

func (w *AMQPConnectionWrapper) Channel() (AMQPChannel, error) {
	ch, err := w.conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (w *AMQPConnectionWrapper) Close() error {
	return w.conn.Close()
}

// CreateAMQPConnection establishes a connection to a RabbitMQ server
// url: The connection string for the RabbitMQ server
// Returns a connection object that can create channels, or an error if connection fails
func CreateAMQPConnection(url string) (AMQPConnection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &AMQPConnectionWrapper{conn: conn}, nil
}

// CloseAMQPConnection gracefully closes an AMQP connection
// conn: The connection to close
// Returns any error encountered during closing
func CloseAMQPConnection(conn AMQPConnectionCloser) error {
	return conn.Close()
}

// CreateAMQPChannel creates a new channel from an existing AMQP connection
// conn: The connection to create a channel from
// Returns a channel object that can be closed, or an error if channel creation fails
func CreateAMQPChannel(conn AMQPChannelCreator) (AMQPChannel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return channel, nil
}

// CloseAMQPChannel gracefully closes an AMQP channel
// channel: The channel to close
// Returns any error encountered during closing
func CloseAMQPChannel(channel AMQPChannelCloser) error {
	return channel.Close()
}

// QueueDeclare creates a queue on the RabbitMQ server if it doesn't exist
// channel: The channel to use for queue declaration
// name: The name of the queue to declare
// Returns the declared queue and any error encountered
func QueueDeclare(channel AMQPQueueDeclarer, name string, durable, autoDelete, exclusive, noWait bool) (amqp.Queue, error) {
	q, err := channel.QueueDeclare(
		name,       // name of the queue
		durable,    // durable: queue will survive broker restart if true
		autoDelete, // delete when unused: queue will be deleted when no consumers if true
		exclusive,  // exclusive: queue can only be used by this connection if true
		noWait,     // no-wait: don't wait for server confirmation if true
		nil,        // arguments for queue declaration
	)

	return q, err
}

func PublishWithContext(ctx context.Context, body []byte, channel AMPQPQueuePublisher, exchange, key string, mandatory, immediate bool) error {
	err := channel.PublishWithContext(ctx,
		exchange,  // exchange
		key,       // routing key
		mandatory, // mandatory
		immediate, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})

	return err
}

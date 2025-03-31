package pkg

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPConnectionCloser defines an interface for closing AMQP connections
type AMQPConnectionCloser interface {
	// Close terminates the connection and all associated resources
	Close() error
}

// AMQPConnection combines connection closing and channel creation capabilities
type AMQPConnection interface {
	AMQPConnectionCloser
	AMQPChannelCreator
}

// AMQPChannelCreator defines an interface for creating AMQP channels
type AMQPChannelCreator interface {
	// Channel creates a new communication channel on the connection
	Channel() (AMQPChannel, error)
}

// AMQPChannelCloser defines an interface for closing AMQP channels
type AMQPChannelCloser interface {
	// Close terminates the channel and all associated resources
	Close() error
}

// AMQPQueuePublisher defines an interface for publishing messages to queues
type AMQPQueuePublisher interface {
	// PublishWithContext sends a message to a specified exchange with context support
	PublishWithContext(ctx context.Context, exchange string, routingKey string, mandatory bool, immediate bool, msg amqp.Publishing) error
}

// AMQPQueueConsumer defines an interface for consuming messages from queues
type AMQPQueueConsumer interface {
	// Consume starts delivering messages from the specified queue
	Consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

// AMQPQueueDeclarer defines an interface for declaring AMQP queues
type AMQPQueueDeclarer interface {
	// QueueDeclare creates or verifies a queue on the RabbitMQ server
	QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp.Table) (amqp.Queue, error)
}

// AMQPChannel combines all the capabilities needed from a channel
type AMQPChannel interface {
	AMQPChannelCloser
	AMQPQueueDeclarer
	AMQPQueuePublisher
	AMQPQueueConsumer
}

// AMQPConnectionWrapper wraps an amqp.Connection to implement AMQPConnection interface
type AMQPConnectionWrapper struct {
	conn *amqp.Connection
}

// Channel creates a new AMQP channel from the wrapped connection
func (w *AMQPConnectionWrapper) Channel() (AMQPChannel, error) {
	ch, err := w.conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// Close terminates the wrapped AMQP connection
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
// durable: If true, queue will survive broker restart
// autoDelete: If true, queue will be deleted when no consumers remain
// exclusive: If true, queue can only be used by this connection
// noWait: If true, don't wait for server confirmation
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

// PublishWithContext publishes a message to a RabbitMQ exchange with context support
// ctx: Context for the operation, allowing for cancellation and timeouts
// body: The message content to publish
// channel: The channel to use for publishing
// exchange: The exchange to publish to
// key: The routing key for the message
// mandatory: If true, server will return an unroutable message
// immediate: If true, server will return a message that cannot be delivered to a consumer immediately
// Returns any error encountered during publishing
func PublishWithContext(ctx context.Context, body []byte, channel AMQPQueuePublisher, exchange, key string, mandatory, immediate bool) error {
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

// Consume starts consuming messages from a RabbitMQ queue
// channel: The channel to use for consuming
// queue: The name of the queue to consume from
// consumer: A unique consumer identifier
// autoAck: If true, server will consider messages delivered once sent
// exclusive: If true, only this consumer can access the queue
// noLocal: If true, server will not send messages published by this connection
// noWait: If true, don't wait for server confirmation
// args: Additional arguments for the consume operation
// Returns a channel of deliveries and any error encountered
func Consume(channel AMQPQueueConsumer, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

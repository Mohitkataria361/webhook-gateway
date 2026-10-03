package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ wraps an AMQP connection and channel with publish/consume helpers.
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New establishes a connection to RabbitMQ, declares the exchange, main queue,
// Dead-Letter Queue, and binds them together.
func New(amqpURL string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp open channel: %w", err)
	}

	r := &RabbitMQ{conn: conn, channel: ch}
	if err := r.setup(); err != nil {
		r.Close()
		return nil, err
	}

	log.Println("[rabbitmq] Connected and topology declared")
	return r, nil
}

// setup declares exchange, DLQ, and main queue with DLQ binding.
func (r *RabbitMQ) setup() error {
	// Declare the topic exchange.
	if err := r.channel.ExchangeDeclare(
		ExchangeName, "topic", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	// Declare the Dead-Letter Queue first.
	if _, err := r.channel.QueueDeclare(
		DLQName, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	// Declare the main delivery queue with DLQ configured as the dead-letter exchange.
	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    "",   // Default exchange
		"x-dead-letter-routing-key": DLQName,
	}
	if _, err := r.channel.QueueDeclare(
		QueueName, true, false, false, false, mainArgs,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	// Bind main queue to the exchange.
	if err := r.channel.QueueBind(
		QueueName, RoutingKey, ExchangeName, false, nil,
	); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}

// Publish sends a WebhookJob to the main exchange.
func (r *RabbitMQ) Publish(ctx context.Context, job *WebhookJob) error {
	body, err := job.Encode()
	if err != nil {
		return fmt.Errorf("encode job: %w", err)
	}

	return r.channel.PublishWithContext(
		ctx,
		ExchangeName, // exchange
		RoutingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Survive broker restarts
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

// PublishToDLQ sends a WebhookJob directly to the Dead-Letter Queue.
func (r *RabbitMQ) PublishToDLQ(ctx context.Context, job *WebhookJob) error {
	body, err := job.Encode()
	if err != nil {
		return fmt.Errorf("encode job for DLQ: %w", err)
	}

	return r.channel.PublishWithContext(
		ctx,
		"",      // default exchange
		DLQName, // routing key = queue name for default exchange
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

// Consume returns a Go channel of AMQP deliveries from the main queue.
// prefetch controls how many unacknowledged messages a worker can hold.
func (r *RabbitMQ) Consume(prefetch int) (<-chan amqp.Delivery, error) {
	if err := r.channel.Qos(prefetch, 0, false); err != nil {
		return nil, fmt.Errorf("set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		QueueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack = false (we ack manually)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("consume queue: %w", err)
	}
	return msgs, nil
}

// Close gracefully shuts down channel and connection.
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

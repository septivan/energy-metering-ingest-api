package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Publisher handles message publishing to RabbitMQ
type Publisher struct {
	conn                  *amqp.Connection
	channel               *amqp.Channel
	confirms              <-chan amqp.Confirmation
	exchange              string
	logger                *zap.Logger
	maxRetries            int
	retryBaseDelay        time.Duration
	publishConfirmTimeout time.Duration
	rabbitMQURL           string
	mu                    sync.Mutex
}

// NewPublisher creates a new RabbitMQ publisher with durable exchange
func NewPublisher(rabbitMQURL, exchange string, maxRetries, retryBaseDelayMs, confirmTimeoutSec int, logger *zap.Logger) (*Publisher, error) {
	p := &Publisher{
		exchange:              exchange,
		logger:                logger,
		maxRetries:            maxRetries,
		retryBaseDelay:        time.Duration(retryBaseDelayMs) * time.Millisecond,
		publishConfirmTimeout: time.Duration(confirmTimeoutSec) * time.Second,
		rabbitMQURL:           rabbitMQURL,
	}

	if err := p.connect(); err != nil {
		return nil, err
	}

	return p, nil
}

// connect establishes connection and channel to RabbitMQ
func (p *Publisher) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close existing connections if any
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}

	conn, err := amqp.Dial(p.rabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Enable publish confirms
	if err := channel.Confirm(false); err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to enable confirm mode: %w", err)
	}

	// Setup confirmation channel
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	p.conn = conn
	p.channel = channel
	p.confirms = confirms

	p.logger.Info("RabbitMQ publisher connected",
		zap.String("exchange", p.exchange),
	)

	return nil
}

// isHealthy checks if connection and channel are open
func (p *Publisher) isHealthy() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || p.conn.IsClosed() {
		return false
	}
	if p.channel == nil {
		return false
	}
	return true
}

// reconnect attempts to reconnect to RabbitMQ
func (p *Publisher) reconnect() error {
	p.logger.Warn("Attempting to reconnect to RabbitMQ")
	return p.connect()
}

// Publish publishes a message with retry logic and confirmation
func (p *Publisher) Publish(ctx context.Context, routingKey string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= p.maxRetries; attempt++ {
		// Check connection health before publishing
		if !p.isHealthy() {
			p.logger.Warn("Connection unhealthy, attempting reconnect",
				zap.Int("attempt", attempt),
			)
			if err := p.reconnect(); err != nil {
				lastErr = fmt.Errorf("reconnect failed: %w", err)
				p.logger.Error("Reconnection failed",
					zap.Int("attempt", attempt),
					zap.Error(err),
				)

				if attempt < p.maxRetries {
					delay := p.retryBaseDelay * time.Duration(1<<uint(attempt-1))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
						continue
					}
				}
				continue
			}
		}

		if err := p.publishWithConfirm(ctx, routingKey, body); err != nil {
			lastErr = err
			p.logger.Warn("Publish attempt failed",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", p.maxRetries),
				zap.Error(err),
			)

			if attempt < p.maxRetries {
				delay := p.retryBaseDelay * time.Duration(1<<uint(attempt-1)) // exponential backoff
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					// Continue to next retry
				}
			}
			continue
		}

		p.logger.Debug("Message published successfully",
			zap.String("routing_key", routingKey),
			zap.Int("attempt", attempt),
		)
		return nil
	}

	return fmt.Errorf("failed to publish after %d attempts: %w", p.maxRetries, lastErr)
}

func (p *Publisher) publishWithConfirm(ctx context.Context, routingKey string, body []byte) error {
	p.mu.Lock()
	channel := p.channel
	confirms := p.confirms
	p.mu.Unlock()

	if channel == nil {
		return fmt.Errorf("channel is nil")
	}

	err := channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	// Wait for confirmation
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("publish not acknowledged by broker")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.publishConfirmTimeout):
		return fmt.Errorf("confirmation timeout")
	}
}

// Close closes the RabbitMQ connection
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			p.logger.Error("Failed to close channel", zap.Error(err))
		}
		p.channel = nil
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			p.logger.Error("Failed to close connection", zap.Error(err))
			return err
		}
		p.conn = nil
	}
	p.logger.Info("RabbitMQ publisher closed")
	return nil
}

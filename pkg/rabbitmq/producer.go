// Package rabbitmq provides RabbitMQ integration for sending email tasks.
package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/reservia/api/pkg/logger"
	"github.com/streadway/amqp"
)

// Producer handles RabbitMQ message publishing.
type Producer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  logger.Logger
}

// Config represents RabbitMQ configuration.
type Config struct {
	URI       string
	QueueName string
}

// EmailTask represents an email task to be sent to the email service.
type EmailTask struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// NewProducer creates a new RabbitMQ producer.
func NewProducer(config Config, logger logger.Logger) (*Producer, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(config.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queue
	_, err = ch.QueueDeclare(
		config.QueueName, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Producer{
		conn:    conn,
		channel: ch,
		logger:  logger,
	}, nil
}

// SendEmailTask sends an email task to the email service.
func (p *Producer) SendEmailTask(queueName string, task EmailTask) error {
	// Marshal the task to JSON
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal email task: %w", err)
	}

	// Publish the message
	err = p.channel.Publish(
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make message persistent
		},
	)
	if err != nil {
		p.logger.Error("Failed to publish email task", "type", task.Type, "error", err)
		return fmt.Errorf("failed to publish email task: %w", err)
	}

	p.logger.Info("Email task sent successfully", "type", task.Type, "queue", queueName)
	return nil
}

// SendAuthEmail sends an authentication email task.
func (p *Producer) SendAuthEmail(queueName, email, code string) error {
	// TODO: Make type employees_auth_email
	task := EmailTask{
		Type: "auth_email",
		Data: map[string]interface{}{
			"email": email,
			"code":  code,
		},
	}
	return p.SendEmailTask(queueName, task)
}

// SendEmployeeRegisterEmail sends an employee registration email task.
func (p *Producer) SendEmployeeRegisterEmail(queueName, email, fullName, employeeID, code string) error {
	task := EmailTask{
		Type: "employee_register_email",
		Data: map[string]interface{}{
			"email":     email,
			"full_name": fullName,
			"id":        employeeID,
			"code":      code,
		},
	}
	return p.SendEmailTask(queueName, task)
}

// Close closes the RabbitMQ connection.
func (p *Producer) Close() error {
	var errs []error

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
	}

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing producer: %v", errs)
	}

	p.logger.Info("RabbitMQ producer closed successfully")
	return nil
}

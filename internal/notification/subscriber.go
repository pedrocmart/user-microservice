package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"user-microservice/internal/models"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type RabbitMQConnection interface {
	Channel() (*amqp.Channel, error)
	Close() error
}

type RabbitMQChannel interface {
	QueueDeclare(queue string, durable, delete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

type RabbitMQSubscriber struct {
	conn           RabbitMQConnection
	channel        RabbitMQChannel
	queueName      string
	logger         *zap.Logger
	handler        EventHandlerInterface
	enableConsumer bool
}

func NewRabbitMQSubscriber(rabbitMQURL, queueName string, logger *zap.Logger, handler EventHandlerInterface, enableConsumer bool) (*RabbitMQSubscriber, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to RabbitMQ")
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RabbitMQ channel")
	}

	return &RabbitMQSubscriber{
		conn:           conn,
		channel:        channel,
		queueName:      queueName,
		logger:         logger.With(zap.String("component", "notification_subscriber")),
		handler:        handler,
		enableConsumer: enableConsumer,
	}, nil
}

func (s *RabbitMQSubscriber) StartConsuming(ctx context.Context) error {
	_, err := s.channel.QueueDeclare(
		s.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare queue")
	}

	s.logger.Info("Queue declared successfully", zap.String("queue", s.queueName))

	err = s.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return errors.Wrap(err, "failed to configure QoS")
	}

	msgs, err := s.channel.Consume(
		s.queueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to register consumer")
	}

	if !s.enableConsumer {
		s.logger.Info("Consumer disabled, not starting message consumption")
		return nil
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("Stopping consumer due to canceled context")
				return
			case msg := <-msgs:
				s.processMessage(ctx, msg)
			}
		}
	}()

	s.logger.Info("Consumer started successfully", zap.String("queue", s.queueName))
	return nil
}

func (s *RabbitMQSubscriber) processMessage(ctx context.Context, msg amqp.Delivery) {
	if msg.Body == nil {
		s.logger.Error("Received empty message", zap.String("type", fmt.Sprintf("%T", msg)))
		// aknowledge even if empty, otherwise it stays stuck in the queue
		if err := msg.Ack(false); err != nil {
			s.logger.Error("Error acknowledging empty message", zap.Error(err))
		}
		return
	}
	s.logger.Debug("Message received", zap.String("body", string(msg.Body)))

	defer func() {
		if err := msg.Ack(false); err != nil {
			s.logger.Error("Error acknowledging message", zap.Error(err))
		} else {
			s.logger.Info("Message acknowledged successfully")
		}
	}()

	var event Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		s.logger.Error("Failed to decode message", zap.Error(err))
		return
	}

	s.logger.Debug("Message received", zap.String("type", event.Type), zap.Time("timestamp", event.Timestamp))

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		s.logger.Error("Failed to serialize payload", zap.Error(err))
		return
	}
	s.logger.Debug("Payload", zap.String("payload", string(payload)))

	switch event.Type {
	case "user.created":
		s.handleUserCreated(ctx, event)
	case "user.updated":
		s.handleUserUpdated(ctx, event)
	case "user.deleted":
		s.handleUserDeleted(ctx, event)
	default:
		s.logger.Warn("Unknown event type", zap.String("type", event.Type))
	}
}

func (s *RabbitMQSubscriber) handleUserCreated(ctx context.Context, event Event) {
	s.logger.Info("Handling user.created event")
	if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
		var user models.User
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			s.logger.Error("Failed to marshal payload", zap.Error(err))
			return
		}

		if err := json.Unmarshal(payloadBytes, &user); err != nil {
			s.logger.Error("Failed to unmarshal payload to user", zap.Error(err))
			return
		}

		s.logger.Debug("User unmarshalled successfully", zap.String("user_id", user.ID))
		if err := s.handler.HandleUserCreated(ctx, &user); err != nil {
			s.logger.Error("Failed to process user.created", zap.Error(err))
		}
	}
}

func (s *RabbitMQSubscriber) handleUserUpdated(ctx context.Context, event Event) {
	s.logger.Info("Handling user.updated event")
	if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
		var user models.User
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			s.logger.Error("Failed to marshal payload", zap.Error(err))
			return
		}

		if err := json.Unmarshal(payloadBytes, &user); err != nil {
			s.logger.Error("Failed to unmarshal payload to user", zap.Error(err))
			return
		}

		s.logger.Debug("User unmarshalled successfully", zap.String("user_id", user.ID))
		if err := s.handler.HandleUserUpdated(ctx, &user); err != nil {
			s.logger.Error("Failed to process user.updated", zap.Error(err))
		}
	}
}

func (s *RabbitMQSubscriber) handleUserDeleted(ctx context.Context, event Event) {
	s.logger.Info("Handling user.deleted event")
	if payload, ok := event.Payload.(map[string]interface{}); ok {
		if id, exists := payload["id"].(string); exists {
			if err := s.handler.HandleUserDeleted(ctx, id); err != nil {
				s.logger.Error("Failed to process user.deleted", zap.Error(err))
			}
		} else {
			s.logger.Error("ID not found or invalid in payload", zap.Any("payload", payload))
		}
	} else {
		s.logger.Error("Payload is not of the expected type", zap.Any("payload", event.Payload))
	}
}

func (s *RabbitMQSubscriber) Close() error {
	if err := s.channel.Close(); err != nil {
		return err
	}
	return s.conn.Close()
}

package notification

import (
	"context"
	"encoding/json"
	"time"

	"user-microservice/internal/models"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type NotificationService interface {
	NotifyUserCreated(ctx context.Context, user *models.User) error
	NotifyUserUpdated(ctx context.Context, user *models.User) error
	NotifyUserDeleted(ctx context.Context, userID string) error
}

type ChannelInterface interface {
	Publish(exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Close() error
}

type RabbitMQNotificationService struct {
	conn      *amqp.Connection
	channel   ChannelInterface
	queueName string
	logger    *zap.Logger
}

func NewRabbitMQNotificationService(rabbitMQURL, queueName string, logger *zap.Logger) (*RabbitMQNotificationService, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to RabbitMQ")
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "error creating RabbitMQ channel")
	}

	_, err = channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, errors.Wrap(err, "error declaring queue")
	}

	return &RabbitMQNotificationService{
		conn:      conn,
		channel:   channel,
		queueName: queueName,
		logger:    logger.With(zap.String("component", "notification_service")),
	}, nil
}

func (s *RabbitMQNotificationService) NotifyUserCreated(ctx context.Context, user *models.User) error {
	event := Event{
		Type:      "user.created",
		Timestamp: time.Now().UTC(),
		Payload:   user,
	}

	return s.sendNotification(ctx, event)
}

func (s *RabbitMQNotificationService) NotifyUserUpdated(ctx context.Context, user *models.User) error {
	event := Event{
		Type:      "user.updated",
		Timestamp: time.Now().UTC(),
		Payload:   user,
	}

	return s.sendNotification(ctx, event)
}

func (s *RabbitMQNotificationService) NotifyUserDeleted(ctx context.Context, userID string) error {
	event := Event{
		Type:      "user.deleted",
		Timestamp: time.Now().UTC(),
		Payload: map[string]string{
			"id": userID,
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *RabbitMQNotificationService) sendNotification(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "error serializing event")
	}

	s.logger.Debug("Payload", zap.String("payload", string(payload)))
	s.logger.Info("Sending message to RabbitMQ", zap.String("queue", s.queueName), zap.String("event_type", event.Type))

	err = s.channel.Publish(
		"",          // Exchange
		s.queueName, // Routing the message to the queue
		false,       // No delivery confirmation
		false,       // No priority
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error sending message to the queue")
	}

	s.logger.Info("Notification sent successfully",
		zap.String("type", event.Type),
		zap.Time("timestamp", event.Timestamp))

	return nil
}

func (s *RabbitMQNotificationService) Close() error {
	if err := s.channel.Close(); err != nil {
		return err
	}
	return s.conn.Close()
}

type MockNotificationService struct {
	logger *zap.Logger
}

func NewMockNotificationService(logger *zap.Logger) *MockNotificationService {
	return &MockNotificationService{
		logger: logger.With(zap.String("component", "mock_notification_service")),
	}
}

func (s *MockNotificationService) NotifyUserCreated(ctx context.Context, user *models.User) error {
	s.logger.Info("Simulating user creation notification", zap.String("id", user.ID))
	return nil
}

func (s *MockNotificationService) NotifyUserUpdated(ctx context.Context, user *models.User) error {
	s.logger.Info("Simulating user update notification", zap.String("id", user.ID))
	return nil
}

func (s *MockNotificationService) NotifyUserDeleted(ctx context.Context, userID string) error {
	s.logger.Info("Simulating user deletion notification", zap.String("id", userID))
	return nil
}

package notification

import (
	"context"
	"errors"
	"testing"
	"user-microservice/internal/models"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockChannel is a mock implementation of ChannelInterface
type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) Publish(exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(exchange, routingKey, mandatory, immediate, msg)
	return args.Error(0)
}

func (m *MockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	callArgs := m.Called(name, durable, autoDelete, exclusive, noWait, args)

	queue, ok := callArgs.Get(0).(amqp.Queue)
	if !ok {
		return amqp.Queue{}, errors.New("failed to assert amqp.Queue from mock arguments")
	}

	return queue, callArgs.Error(1)
}

func (m *MockChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRabbitMQNotificationService_NotifyUserCreated(t *testing.T) {
	mockChannel := new(MockChannel)
	mockChannel.On("Publish", "", "testQueue", false, false, mock.Anything).Return(nil)

	logger, _ := zap.NewDevelopment()

	service := &RabbitMQNotificationService{
		conn:      nil, // Not needed for this test
		channel:   mockChannel,
		queueName: "testQueue",
		logger:    logger,
	}

	user := &models.User{
		ID:        uuid.New().String(),
		FirstName: "John Doe",
	}

	err := service.NotifyUserCreated(context.Background(), user)

	assert.NoError(t, err)

	mockChannel.AssertExpectations(t)
}

func TestRabbitMQNotificationService_NotifyUserUpdated(t *testing.T) {
	mockChannel := new(MockChannel)
	mockChannel.On("Publish", "", "testQueue", false, false, mock.Anything).Return(nil)

	logger, _ := zap.NewDevelopment()

	service := &RabbitMQNotificationService{
		conn:      nil, // Not needed for this test
		channel:   mockChannel,
		queueName: "testQueue",
		logger:    logger,
	}

	user := &models.User{
		ID:        uuid.New().String(),
		FirstName: "John Doe",
	}

	err := service.NotifyUserUpdated(context.Background(), user)

	assert.NoError(t, err)

	mockChannel.AssertExpectations(t)
}

func TestRabbitMQNotificationService_NotifyUserDeleted(t *testing.T) {
	mockChannel := new(MockChannel)
	mockChannel.On("Publish", "", "testQueue", false, false, mock.Anything).Return(nil)

	logger, _ := zap.NewDevelopment()

	service := &RabbitMQNotificationService{
		conn:      nil, // Not needed for this test
		channel:   mockChannel,
		queueName: "testQueue",
		logger:    logger,
	}

	userID := uuid.New().String()

	err := service.NotifyUserDeleted(context.Background(), userID)

	assert.NoError(t, err)

	mockChannel.AssertExpectations(t)
}

func TestMockNotificationService_NotifyUserCreated(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	service := NewMockNotificationService(logger)

	user := &models.User{
		ID:        uuid.New().String(),
		FirstName: "John Doe",
	}

	err := service.NotifyUserCreated(context.Background(), user)

	assert.NoError(t, err)
}

func TestMockNotificationService_NotifyUserUpdated(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	service := NewMockNotificationService(logger)

	user := &models.User{
		ID:        uuid.New().String(),
		FirstName: "John Doe",
	}

	err := service.NotifyUserUpdated(context.Background(), user)

	assert.NoError(t, err)
}

func TestMockNotificationService_NotifyUserDeleted(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	service := NewMockNotificationService(logger)

	userID := uuid.New().String()

	err := service.NotifyUserDeleted(context.Background(), userID)

	assert.NoError(t, err)
}

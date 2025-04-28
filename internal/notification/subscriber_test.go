package notification

import (
	"context"
	"errors"
	"testing"
	"user-microservice/internal/models"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mocking RabbitMQConnection
type MockRabbitMQConnection struct {
	mock.Mock
}

func (m *MockRabbitMQConnection) Channel() (*amqp.Channel, error) {
	args := m.Called()
	return args.Get(0).(*amqp.Channel), args.Error(1)
}

func (m *MockRabbitMQConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mocking RabbitMQChannel
type MockRabbitMQChannel struct {
	mock.Mock
}

func (m *MockRabbitMQChannel) QueueDeclare(queue string, durable, delete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	argsC := m.Called(queue, durable, delete, exclusive, noWait, args)
	return argsC.Get(0).(amqp.Queue), argsC.Error(1)
}

func (m *MockRabbitMQChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	args := m.Called(prefetchCount, prefetchSize, global)
	return args.Error(0)
}

func (m *MockRabbitMQChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	argsC := m.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	return argsC.Get(0).(<-chan amqp.Delivery), argsC.Error(1)
}

func (m *MockRabbitMQChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mocking EventHandlerInterface (for testing)
type MockEventHandler struct {
	mock.Mock
}

func (m *MockEventHandler) HandleUserCreated(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockEventHandler) HandleUserUpdated(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockEventHandler) HandleUserDeleted(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestRabbitMQSubscriber_StartConsuming(t *testing.T) {
	mockConn := new(MockRabbitMQConnection)
	mockChannel := new(MockRabbitMQChannel)
	mockHandler := new(MockEventHandler)

	mockChannel.On("QueueDeclare", "test-queue", true, false, false, false, mock.Anything).Return(amqp.Queue{}, nil).Once()
	mockChannel.On("Qos", 1, 0, false).Return(nil).Once()

	mockChannel.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).Return(make(<-chan amqp.Delivery), nil).Once()

	logger, _ := zap.NewProduction()
	subscriber := &RabbitMQSubscriber{
		conn:           mockConn,
		channel:        mockChannel,
		queueName:      "test-queue",
		logger:         logger,
		handler:        mockHandler,
		enableConsumer: true,
	}

	ctx := context.Background()
	err := subscriber.StartConsuming(ctx)

	assert.NoError(t, err)

	mockConn.AssertExpectations(t)
	mockChannel.AssertExpectations(t)
}

func TestRabbitMQSubscriber_StartConsuming_WithError(t *testing.T) {
	mockConn := new(MockRabbitMQConnection)
	mockChannel := new(MockRabbitMQChannel)
	mockHandler := new(MockEventHandler)

	mockChannel.On("QueueDeclare", "test-queue", true, false, false, false, mock.Anything).Return(amqp.Queue{}, nil).Once()
	mockChannel.On("Qos", 1, 0, false).Return(nil).Once()
	mockChannel.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).Return(make(<-chan amqp.Delivery), errors.New("failed to register consumer")).Once()

	logger, _ := zap.NewProduction()
	subscriber := &RabbitMQSubscriber{
		conn:           mockConn,
		channel:        mockChannel,
		queueName:      "test-queue",
		logger:         logger,
		handler:        mockHandler,
		enableConsumer: true,
	}

	ctx := context.Background()
	err := subscriber.StartConsuming(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register consumer")

	mockConn.AssertExpectations(t)
	mockChannel.AssertExpectations(t)
}

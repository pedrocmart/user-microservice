package notification

import (
	"context"
	"user-microservice/internal/models"

	"go.uber.org/zap"
)

type EventHandlerInterface interface {
	HandleUserCreated(ctx context.Context, user *models.User) error
	HandleUserUpdated(ctx context.Context, user *models.User) error
	HandleUserDeleted(ctx context.Context, userID string) error
}

type EventHandler struct {
	logger *zap.Logger
}

func NewEventHandler(logger *zap.Logger) *EventHandler {
	return &EventHandler{logger: logger}
}

func (h *EventHandler) HandleUserCreated(ctx context.Context, user *models.User) error {
	h.logger.Info("Processing user.created event", zap.String("id", user.ID))
	return nil
}

func (h *EventHandler) HandleUserUpdated(ctx context.Context, user *models.User) error {
	h.logger.Info("Processing user.updated event", zap.String("id", user.ID))
	return nil
}

func (h *EventHandler) HandleUserDeleted(ctx context.Context, userID string) error {
	h.logger.Info("Processing user.deleted event", zap.String("id", userID))
	return nil
}

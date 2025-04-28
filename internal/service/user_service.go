package service

import (
	"context"
	"fmt"
	"time"

	"user-microservice/internal/models"
	"user-microservice/internal/notification"
	"user-microservice/internal/repository"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrInvalidInput          = errors.New("invalid input data")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrNicknameAlreadyExists = errors.New("nickname already registered")
	ErrUserNotFound          = repository.ErrUserNotFound
)

const (
	notificationTimeout = 5 * time.Second
)

type UserServiceInterface interface {
	CreateUser(ctx context.Context, firstName, lastName, nickname, password, email, country string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id, firstName, lastName, nickname, email, country string) (*models.User, error)
	UpdatePassword(ctx context.Context, id, password string) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, country, email, nickname, firstname, lastname string, page, pageSize int) ([]*models.User, int, error)
}

type UserService struct {
	repo         repository.UserRepository
	notification notification.NotificationService
	logger       *zap.Logger
}

func NewUserService(repo repository.UserRepository, notification notification.NotificationService, logger *zap.Logger) *UserService {
	return &UserService{
		repo:         repo,
		notification: notification,
		logger:       logger.With(zap.String("component", "user_service")),
	}
}

func (s *UserService) CreateUser(ctx context.Context, firstName, lastName, nickname, password, email, country string) (*models.User, error) {
	user, err := models.NewUser(firstName, lastName, nickname, password, email, country)
	if err != nil {
		return nil, errors.Wrap(err, "error creating user")
	}

	_, err = s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, errors.Wrap(err, "error checking existing email")
	}

	_, err = s.repo.GetByNickname(ctx, nickname)
	if err == nil {
		return nil, ErrNicknameAlreadyExists
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, errors.Wrap(err, "error checking existing nickname")
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, "error persisting user")
	}

	s.sendNotification(ctx, func(ctx context.Context) error {
		if err := s.notification.NotifyUserCreated(ctx, user); err != nil {
			fmt.Println("Error notifying user create:", err)
		}
		return nil
	})

	user.SanitizeForOutput()
	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
		return nil, errors.Wrap(err, "error fetching user")
	}

	//make sure we dont return the password
	user.SanitizeForOutput()
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id, firstName, lastName, nickname, email, country string) (*models.User, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching user for update")
	}

	if err := s.validateAndCheckEmail(ctx, user, email); err != nil {
		return nil, errors.Wrap(err, "error validating email")
	}

	if err := s.validateAndCheckNickname(ctx, user, nickname); err != nil {
		return nil, errors.Wrap(err, "error validating nickname")
	}

	if err := user.Update(firstName, lastName, nickname, email, country); err != nil {
		return nil, errors.Wrap(err, "error updating user fields")
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, errors.Wrap(err, "error updating user")
	}

	s.sendNotification(ctx, func(ctx context.Context) error {
		if err := s.notification.NotifyUserUpdated(ctx, user); err != nil {
			fmt.Println("Error notifying user update:", err)
		}
		return nil
	})

	user.SanitizeForOutput()
	return user, nil
}

func (s *UserService) validateAndCheckEmail(ctx context.Context, user *models.User, email string) error {
	if email != "" && email != user.Email {
		if err := user.ValidateEmail(email); err != nil {
			return err
		}

		_, err := s.repo.GetByEmail(ctx, email)
		if err == nil {
			return ErrEmailAlreadyExists
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return errors.Wrap(err, "error checking existing email")
		}
	}
	return nil
}

func (s *UserService) validateAndCheckNickname(ctx context.Context, user *models.User, nickname string) error {
	if nickname != "" && nickname != user.Nickname {
		_, err := s.repo.GetByNickname(ctx, nickname)
		if err == nil {
			return ErrNicknameAlreadyExists
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return errors.Wrap(err, "error checking existing nickname")
		}
	}
	return nil
}

func (s *UserService) UpdatePassword(ctx context.Context, id, password string) error {
	if id == "" || password == "" {
		return ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "error fetching user for password update")
	}

	if err := user.UpdatePassword(password); err != nil {
		return errors.Wrap(err, "error updating password")
	}

	if err := s.repo.UpdatePassword(ctx, id, user.Password); err != nil {
		return errors.Wrap(err, "error persisting password update")
	}

	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "error fetching user for deletion")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "error removing user")
	}

	s.sendNotification(ctx, func(ctx context.Context) error {
		if err := s.notification.NotifyUserDeleted(ctx, id); err != nil {
			fmt.Println("Error notifying user create:", err)
		}
		return nil
	})

	s.logger.Info("user removed successfully",
		zap.String("id", id),
		zap.String("email", user.Email))

	return nil
}

func (s *UserService) ListUsers(ctx context.Context, country, email, nickname, firstname, lastname string, page, pageSize int) ([]*models.User, int, error) {
	filter := repository.FilterOptions{
		FirstName: firstname,
		LastName:  lastname,
		Country:   country,
		Email:     email,
		Nickname:  nickname,
	}

	pagination := repository.PaginationOptions{
		Page:     page,
		PageSize: pageSize,
	}

	users, total, err := s.repo.List(ctx, filter, pagination)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error listing users")
	}

	// ensure no passwords are returned
	for _, user := range users {
		user.SanitizeForOutput()
	}

	return users, total, nil
}

func (s *UserService) sendNotification(ctx context.Context, fn func(context.Context) error) {
	if s.notification == nil {
		return
	}

	go func() {
		notifyCtx, cancel := context.WithTimeout(ctx, notificationTimeout)
		defer cancel()

		if err := fn(notifyCtx); err != nil {
			s.logger.Error("notification failed", zap.Error(err))
		}
	}()
}

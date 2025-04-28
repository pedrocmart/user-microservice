package service_test

import (
	"context"
	"testing"
	"time"

	"user-microservice/internal/models"
	"user-microservice/internal/repository"
	"user-microservice/internal/service"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository is a mock of the user repository for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByNickname(ctx context.Context, nickname string) (*models.User, error) {
	args := m.Called(ctx, nickname)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id, password string) error {
	args := m.Called(ctx, id, password)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, filter repository.FilterOptions, pagination repository.PaginationOptions) ([]*models.User, int, error) {
	args := m.Called(ctx, filter, pagination)

	// safely assert the type of the first argument
	if users, ok := args.Get(0).([]*models.User); ok {
		return users, args.Int(1), args.Error(2)
	}

	return nil, 0, args.Error(2)
}

// MockNotificationService is a mock of the notification service for testing
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) NotifyUserCreated(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyUserUpdated(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockNotificationService) NotifyUserDeleted(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func setupTest(t *testing.T) (*zap.Logger, *MockUserRepository, *MockNotificationService) {
	logger := zaptest.NewLogger(t)
	mockRepo := new(MockUserRepository)
	mockNotification := new(MockNotificationService)
	return logger, mockRepo, mockNotification
}

func TestUserService_CreateUser(t *testing.T) {
	logger, mockRepo, mockNotification := setupTest(t)

	userService := service.NewUserService(mockRepo, mockNotification, logger)

	firstName := "John"
	lastName := "Travolta"
	nickname := "John123"
	password := "password123"
	email := "john@gggmail.com"
	country := "us"

	t.Run("successful creation", func(t *testing.T) {
		mockRepo.On("GetByEmail", mock.Anything, email).Return(nil, repository.ErrUserNotFound).Once()
		mockRepo.On("GetByNickname", mock.Anything, nickname).Return(nil, repository.ErrUserNotFound).Once()

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
			return u.FirstName == firstName &&
				u.LastName == lastName &&
				u.Nickname == nickname &&
				u.Email == email &&
				u.Country == country
		})).Return(nil).Once()

		mockNotification.On("NotifyUserCreated", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

		user, err := userService.CreateUser(context.Background(), firstName, lastName, nickname, password, email, country)

		// Check results
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, firstName, user.FirstName)
		assert.Equal(t, lastName, user.LastName)
		assert.Equal(t, nickname, user.Nickname)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, country, user.Country)
		assert.Empty(t, user.Password) // Password should not be returned

		mockRepo.AssertExpectations(t)

		// we don't verify mockNotification because it's called in a goroutine
	})

	// Test case: email already exists
	t.Run("email already exists", func(t *testing.T) {
		existingUser := &models.User{
			ID:        uuid.New().String(),
			FirstName: "Existing",
			LastName:  "User",
			Nickname:  "EU456",
			Email:     email,
			Country:   "US",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		mockRepo.On("GetByEmail", mock.Anything, email).Return(existingUser, nil).Once()

		user, err := userService.CreateUser(context.Background(), firstName, lastName, nickname, password, email, country)

		// Check results
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, service.ErrEmailAlreadyExists, err)

		mockRepo.AssertExpectations(t)
	})

	// Test case: nickname already exists
	t.Run("nickname already exists", func(t *testing.T) {
		existingUser := &models.User{
			ID:        uuid.New().String(),
			FirstName: "Existing",
			LastName:  "User",
			Nickname:  nickname,
			Email:     "different@example.com",
			Country:   "US",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		mockRepo.On("GetByEmail", mock.Anything, email).Return(nil, repository.ErrUserNotFound).Once()
		mockRepo.On("GetByNickname", mock.Anything, nickname).Return(existingUser, nil).Once()

		user, err := userService.CreateUser(context.Background(), firstName, lastName, nickname, password, email, country)

		// Check results
		assert.Nil(t, user)
		assert.Equal(t, service.ErrNicknameAlreadyExists, err)

		mockRepo.AssertExpectations(t)
	})

	// Test case: invalid data
	t.Run("invalid data", func(t *testing.T) {
		user, err := userService.CreateUser(context.Background(), "", lastName, nickname, password, email, country)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "first name is required")
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	logger, mockRepo, mockNotification := setupTest(t)

	userService := service.NewUserService(mockRepo, mockNotification, logger)

	userID := uuid.New().String()
	existingUser := &models.User{
		ID:        userID,
		FirstName: "John",
		LastName:  "Travolta",
		Nickname:  "John123",
		Password:  "hashedpassword",
		Email:     "john@gggmail.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Test case: user found
	t.Run("user found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(existingUser, nil).Once()

		user, err := userService.GetUserByID(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, existingUser.FirstName, user.FirstName)
		assert.Equal(t, existingUser.LastName, user.LastName)
		assert.Equal(t, existingUser.Nickname, user.Nickname)
		assert.Equal(t, existingUser.Email, user.Email)
		assert.Equal(t, existingUser.Country, user.Country)
		assert.Empty(t, user.Password) // Password should not be returned

		mockRepo.AssertExpectations(t)
	})

	// Test case: user not found
	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, "non-existent-id").Return(nil, repository.ErrUserNotFound).Once()

		user, err := userService.GetUserByID(context.Background(), "non-existent-id")

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, repository.ErrUserNotFound))

		mockRepo.AssertExpectations(t)
	})

	// Test case: invalid ID
	t.Run("invalid ID", func(t *testing.T) {
		user, err := userService.GetUserByID(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, service.ErrInvalidInput, err)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	logger, mockRepo, mockNotification := setupTest(t)

	userService := service.NewUserService(mockRepo, mockNotification, logger)

	userID := uuid.New().String()
	existingUser := &models.User{
		ID:        userID,
		FirstName: "John",
		LastName:  "Travolta",
		Nickname:  "John123",
		Password:  "hashedpassword",
		Email:     "john@gggmail.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Test case: successful removal
	t.Run("successful removal", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(existingUser, nil).Once()
		mockRepo.On("Delete", mock.Anything, userID).Return(nil).Once()

		mockNotification.On("NotifyUserDeleted", mock.Anything, userID).Return(nil)

		err := userService.DeleteUser(context.Background(), userID)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	// Test case: user not found
	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, "non-existent-id").Return(nil, repository.ErrUserNotFound).Once()

		err := userService.DeleteUser(context.Background(), "non-existent-id")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repository.ErrUserNotFound))

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	logger, mockRepo, mockNotification := setupTest(t)
	userService := service.NewUserService(mockRepo, mockNotification, logger)

	userID := uuid.New().String()
	firstName := "John"
	lastName := "Travolta"
	nickname := "John123"
	email := "john@gggmail.com"
	country := "US"

	// Test case: successful update
	t.Run("successful update", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(&models.User{
			ID:        userID,
			FirstName: "John",
			LastName:  "Travolta",
			Nickname:  "John1234",
			Email:     "john@travolta.com",
			Country:   "UK",
		}, nil).Once()

		mockRepo.On("GetByEmail", mock.Anything, email).Return(nil, repository.ErrUserNotFound).Once()
		mockRepo.On("GetByNickname", mock.Anything, nickname).Return(nil, repository.ErrUserNotFound).Once()

		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
			return u.FirstName == firstName && u.LastName == lastName &&
				u.Nickname == nickname && u.Email == email && u.Country == country
		})).Return(nil).Once()

		mockNotification.On("NotifyUserUpdated", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

		user, err := userService.UpdateUser(context.Background(), userID, firstName, lastName, nickname, email, country)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, firstName, user.FirstName)
		assert.Equal(t, lastName, user.LastName)
		assert.Equal(t, nickname, user.Nickname)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, country, user.Country)

		mockRepo.AssertExpectations(t)
	})

	// Test case: user not found
	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(nil, repository.ErrUserNotFound).Once()

		user, err := userService.UpdateUser(context.Background(), userID, firstName, lastName, nickname, email, country)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	logger, mockRepo, _ := setupTest(t)
	userService := service.NewUserService(mockRepo, nil, logger)

	userID := uuid.New().String()
	newPassword := "newsecurepassword"

	// Test case: successful password update
	t.Run("successful password update", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(&models.User{
			ID:        userID,
			FirstName: "John",
			LastName:  "Travolta",
			Nickname:  "John123",
			Email:     "john@gggmail.com",
			Country:   "US",
		}, nil).Once()

		mockRepo.On("UpdatePassword", mock.Anything, userID, mock.MatchedBy(func(pwd string) bool {
			return bcrypt.CompareHashAndPassword([]byte(pwd), []byte(newPassword)) == nil
		})).Return(nil).Once()

		err := userService.UpdatePassword(context.Background(), userID, newPassword)

		assert.NoError(t, err)
	})

	// Test case: user not found
	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, userID).Return(nil, repository.ErrUserNotFound).Once()

		err := userService.UpdatePassword(context.Background(), userID, newPassword)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_ListUsers(t *testing.T) {
	logger, mockRepo, _ := setupTest(t)
	userService := service.NewUserService(mockRepo, nil, logger)

	page := 1
	pageSize := 10
	country := "US"

	// Test case: successful listing
	t.Run("successful listing", func(t *testing.T) {
		mockRepo.On("List", mock.Anything, repository.FilterOptions{Country: country}, repository.PaginationOptions{Page: page, PageSize: pageSize}).
			Return([]*models.User{
				{ID: uuid.New().String(), FirstName: "John", LastName: "Travolta", Nickname: "john123", Email: "john@gggmail.com", Country: country},
			}, 1, nil).Once()

		users, total, err := userService.ListUsers(context.Background(), country, "", "", "", "", page, pageSize)

		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, 1, total)
	})

	// Test case: error while listing
	t.Run("error while listing", func(t *testing.T) {
		mockRepo.On("List", mock.Anything, repository.FilterOptions{Country: country}, repository.PaginationOptions{Page: page, PageSize: pageSize}).
			Return(nil, 0, errors.New("error listing users")).Once()

		users, total, err := userService.ListUsers(context.Background(), country, "", "", "", "", page, pageSize)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, 0, total)
	})
}

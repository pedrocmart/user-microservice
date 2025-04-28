package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-microservice/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, firstName, lastName, nickname, password, email, country string) (*models.User, error) {
	args := m.Called(ctx, firstName, lastName, nickname, password, email, country)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id, firstName, lastName, nickname, email, country string) (*models.User, error) {
	args := m.Called(ctx, id, firstName, lastName, nickname, email, country)
	if user, ok := args.Get(0).(*models.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockUserService) UpdatePassword(ctx context.Context, id, password string) error {
	args := m.Called(ctx, id, password)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) ListUsers(ctx context.Context, country, email, nickname, firstname, lastname string, page, pageSize int) ([]*models.User, int, error) {
	args := m.Called(ctx, country, email, nickname, firstname, lastname, page, pageSize)
	if users, ok := args.Get(0).([]*models.User); ok {
		return users, args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func TestCreateUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	reqBody := CreateUserRequest{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "jdoe",
		Password:  "password",
		Email:     "john@example.com",
		Country:   "USA",
	}

	user := &models.User{
		ID:    "123",
		Email: "john@example.com",
	}

	mockService.On("CreateUser", mock.Anything, reqBody.FirstName, reqBody.LastName, reqBody.Nickname, reqBody.Password, reqBody.Email, reqBody.Country).
		Return(user, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateUser_InvalidBody(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte(`invalid json`)))
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	user := &models.User{
		ID:    "123",
		Email: "john@example.com",
	}

	mockService.On("GetUserByID", mock.Anything, "123").Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	reqCtx := chi.NewRouteContext()
	reqCtx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, reqCtx))

	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetUser_MissingID(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	reqCtx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, reqCtx))

	w := httptest.NewRecorder()
	handler.GetUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	reqBody := UpdateUserRequest{
		FirstName: "Updated",
		LastName:  "User",
		Nickname:  "updateduser",
		Email:     "updated@example.com",
		Country:   "UK",
	}

	user := map[string]interface{}{"id": "123", "email": "updated@example.com"}

	mockService.On("UpdateUser", mock.Anything, "123", reqBody.FirstName, reqBody.LastName, reqBody.Nickname, reqBody.Email, reqBody.Country).Return(user, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/users/123", bytes.NewReader(body))
	reqCtx := chi.NewRouteContext()
	reqCtx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, reqCtx))

	w := httptest.NewRecorder()
	handler.UpdateUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	userID := "123"

	mockService.On("DeleteUser", mock.Anything, userID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/123", nil)
	reqCtx := chi.NewRouteContext()
	reqCtx.URLParams.Add("id", userID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, reqCtx))

	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestListUsers_Success(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	users := []*models.User{
		{ID: "123", Email: "john@example.com"},
		{ID: "124", Email: "jane@example.com"},
	}
	total := 2

	mockService.On("ListUsers", mock.Anything, "", "", "", "", "", 1, 10).Return(users, total, nil)

	req := httptest.NewRequest(http.MethodGet, "/users?page=1&page_size=10", nil)

	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response ListUsersResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, total, response.TotalCount)

	assert.Equal(t, len(users), len(response.Users))

	mockService.AssertExpectations(t)
}

func TestUpdatePassword(t *testing.T) {
	mockService := new(MockUserService)
	logger := zap.NewNop()
	handler := NewUserHandler(mockService, logger)

	reqBody := UpdatePasswordRequest{
		Password: "newpassword",
	}

	userID := "123"

	mockService.On("UpdatePassword", mock.Anything, userID, reqBody.Password).Return(nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/users/123/password", bytes.NewReader(body))
	reqCtx := chi.NewRouteContext()
	reqCtx.URLParams.Add("id", userID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, reqCtx))

	w := httptest.NewRecorder()
	handler.UpdatePassword(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockService.AssertExpectations(t)
}

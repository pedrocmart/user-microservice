package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"user-microservice/internal/models"
	"user-microservice/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// UserHandler manages HTTP requests related to users
type UserHandler struct {
	service service.UserServiceInterface
	logger  *zap.Logger
}

// NewUserHandler creates a new instance of UserHandler
func NewUserHandler(service service.UserServiceInterface, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger.With(zap.String("component", "user_handler")),
	}
}

// RegisterRoutes registers the handler routes on the router
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)
		r.Post("/", h.CreateUser)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetUser)
			r.Put("/", h.UpdateUser)
			r.Delete("/", h.DeleteUser)
			r.Put("/password", h.UpdatePassword)
		})
	})
}

// CreateUserRequest represents the body of the request to create a user
type CreateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname"`
	Password  string `json:"password"`
	Email     string `json:"email"`
	Country   string `json:"country"`
}

// UpdateUserRequest represents the body of the request to update a user
type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Country   string `json:"country"`
}

// UpdatePasswordRequest represents the body of the request to update a password
type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []*models.User `json:"users"`
	TotalCount int            `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
}

// respondWithJSON sends a JSON response
func (h *UserHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("error serializing response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// respondWithError sends an error response
func (h *UserHandler) respondWithError(w http.ResponseWriter, code int, err error) {
	h.logger.Error("error in request",
		zap.Int("status", code),
		zap.Error(err))

	// Map known errors to appropriate HTTP status codes
	if errors.Is(err, service.ErrInvalidInput) {
		code = http.StatusBadRequest
	} else if errors.Is(err, service.ErrEmailAlreadyExists) ||
		errors.Is(err, service.ErrNicknameAlreadyExists) {
		code = http.StatusConflict
	} else if errors.Is(err, service.ErrUserNotFound) {
		code = http.StatusNotFound
	}

	h.respondWithJSON(w, code, ErrorResponse{Error: err.Error()})
}

// @Summary: Create a new user
// @Description: Create a new user with the provided details
// @Tags: users
// @Accept: json
// @Produce: json
// @Param user body CreateUserRequest true "User details"
// @Success 201 {object} models.User
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	user, err := h.service.CreateUser(
		r.Context(),
		req.FirstName,
		req.LastName,
		req.Nickname,
		req.Password,
		req.Email,
		req.Country,
	)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, user)
}

// @Summary: Get a user by ID
// @Description: Retrieve a user by their ID
// @Tags: users
// @Produce: json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("ID is required"))
		return
	}

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

// @Summary: Update a user by ID
// @Description: Update a user's details by their ID
// @Tags: users
// @Accept: json
// @Produce: json
// @Param id path string true "User ID"
// @Param user body UpdateUserRequest true "User details"
// @Success 200 {object} models.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("ID is required"))
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	user, err := h.service.UpdateUser(
		r.Context(),
		id,
		req.FirstName,
		req.LastName,
		req.Nickname,
		req.Email,
		req.Country,
	)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

// @Summary: Update a user's password
// @Description: Update a user's password by their ID
// @Tags: users
// @Accept: json
// @Produce: json
// @Param id path string true "User ID"
// @Param password body UpdatePasswordRequest true "New password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/password [put]
func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("ID is required"))
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	if err := h.service.UpdatePassword(r.Context(), id, req.Password); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

// @Summary: Delete a user by ID
// @Description: Delete a user by their ID
// @Tags: users
// @Produce: json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondWithError(w, http.StatusBadRequest, errors.New("ID is required"))
		return
	}

	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "user removed successfully"})
}

// @Summary: List users
// @Description: Retrieve a list of users with optional filters and pagination
// @Tags: users
// @Produce: json
// @Param country query string false "Country"
// @Param nickname query string false "Nickname"
// @Param lastname query string false "Last name"
// @Param email query string false "Email"
// @Param firstname query string false "First name"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} ListUsersResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Filter parameters
	country := r.URL.Query().Get("country")
	nickname := r.URL.Query().Get("nickname")
	lastname := r.URL.Query().Get("lastname")
	email := r.URL.Query().Get("email")
	firstname := r.URL.Query().Get("firstname")

	// Pagination parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 10

	if pageStr != "" {
		pageInt, err := strconv.Atoi(pageStr)
		if err == nil && pageInt > 0 {
			page = pageInt
		}
	}

	if pageSizeStr != "" {
		pageSizeInt, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSizeInt > 0 && pageSizeInt <= 100 {
			pageSize = pageSizeInt
		}
	}

	users, total, err := h.service.ListUsers(r.Context(), country, email, nickname, firstname, lastname, page, pageSize)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	response := ListUsersResponse{
		Users:      users,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

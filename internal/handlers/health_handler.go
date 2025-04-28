package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"user-microservice/internal/repository"

	"user-microservice/internal/config"

	"go.uber.org/zap"
)

type HealthHandler struct {
	db        repository.HealthChecker
	logger    *zap.Logger
	appConfig *config.AppConfig
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

func NewHealthHandler(db repository.HealthChecker, logger *zap.Logger, appConfig *config.AppConfig) *HealthHandler {
	return &HealthHandler{
		db:        db,
		logger:    logger.With(zap.String("component", "health_handler")),
		appConfig: appConfig,
	}
}

func (h *HealthHandler) RegisterRoutes(r http.Handler) {
	if router, ok := r.(interface {
		Get(pattern string, handlerFn http.HandlerFunc)
	}); ok {
		router.Get("/health", h.HealthCheck)
		router.Get("/readiness", h.ReadinessCheck)
	}
}

// HealthCheck checks the overall health of the service
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	services := map[string]string{
		h.appConfig.Name: "ok",
	}

	// Check database connection
	if err := h.db.CheckHealth(); err != nil {
		h.logger.Error("database health check failed", zap.Error(err))
		status = "degraded"
		services["database"] = "error: " + err.Error()
	} else {
		services["database"] = "ok"
	}

	response := HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Version:   h.appConfig.Version,
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("error serializing health check response", zap.Error(err))
	}
}

// ReadinessCheck checks if the service is ready to receive traffic
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := h.db.CheckHealth(); err != nil {
		h.logger.Error("service is not ready - database failure", zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

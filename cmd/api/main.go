package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-microservice/internal/config"
	"user-microservice/internal/handlers"
	"user-microservice/internal/migration"
	"user-microservice/internal/notification"
	"user-microservice/internal/repository"
	"user-microservice/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	_ "user-microservice/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title User Management Microservice API
func main() {
	if err := run(); err != nil {
		fmt.Printf("service exited with error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Initialize logger
	logger, err := cfg.Logging.NewLogger()
	if err != nil {
		return fmt.Errorf("error initializing logger: %w", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("error syncing logger", zap.Error(err))
		}
	}()

	logger.Info("Starting user microservice", zap.String("service", cfg.App.Name), zap.String("version", cfg.App.Version))

	// Database setup
	db, err := sqlx.Connect("postgres", cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify database connection
	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	// Run migrations
	if err := migration.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	logger.Info("Database migrations applied successfully")

	// Initialize repositories
	userRepo := repository.NewPostgresUserRepository(db, logger)

	// Initialize notification service
	notificationSvc, cleanup, err := setupNotificationService(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize notification service: %w", err)
	}
	defer cleanup()

	eventHandler := notification.NewEventHandler(logger)

	// Set up RabbitMQ subscriber
	subscriber, subscriberCleanup, err := setupRabbitMQSubscriber(cfg, logger, eventHandler)
	if err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ subscriber: %w", err)
	}
	defer subscriberCleanup()

	// Initialize services with notification dependency
	userService := service.NewUserService(userRepo, notificationSvc, logger)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService, logger)
	healthHandler := handlers.NewHealthHandler(userRepo, logger, &cfg.App)

	// Set up HTTP server
	server := setupHTTPServer(cfg, userHandler, healthHandler, logger)

	// Using errgroup to manage all goroutines
	g, ctx := errgroup.WithContext(context.Background())

	// Start HTTP server in goroutine
	g.Go(func() error {
		logger.Info("HTTP server starting", zap.Int("port", cfg.Server.Port), zap.String("environment", cfg.App.Environment))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil
	})

	// Start RabbitMQ subscriber in goroutine
	if subscriber != nil {
		g.Go(func() error {
			logger.Info("Starting RabbitMQ subscriber")
			return subscriber.StartConsuming(ctx)
		})
	}

	// Handle graceful shutdown
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-quit:
			logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
		}

		// Create shutdown context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		logger.Info("Server exited gracefully")
		return nil
	})

	// Wait for all goroutines to finish
	return g.Wait()
}

func setupNotificationService(cfg *config.Config, logger *zap.Logger) (notification.NotificationService, func(), error) {
	if cfg.Notification.RabbitMQURL == "" || cfg.Notification.QueueName == "" {
		logger.Warn("RabbitMQ configuration missing, using mock notification service")
		return notification.NewMockNotificationService(logger), func() {}, nil
	}

	rabbitSvc, err := notification.NewRabbitMQNotificationService(
		cfg.Notification.RabbitMQURL,
		cfg.Notification.QueueName,
		logger,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RabbitMQ service: %w", err)
	}

	cleanup := func() {
		logger.Info("Closing RabbitMQ connection")
		if err := rabbitSvc.Close(); err != nil {
			logger.Error("Error closing RabbitMQ connection", zap.Error(err))
		}
	}

	logger.Info("RabbitMQ notification service initialized")
	return rabbitSvc, cleanup, nil
}

func setupHTTPServer(cfg *config.Config, userHandler *handlers.UserHandler, healthHandler *handlers.HealthHandler, logger *zap.Logger) *http.Server {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	// Routes
	userHandler.RegisterRoutes(r)
	healthHandler.RegisterRoutes(r)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func setupRabbitMQSubscriber(cfg *config.Config, logger *zap.Logger, handler notification.EventHandlerInterface) (*notification.RabbitMQSubscriber, func(), error) {
	if cfg.Notification.RabbitMQURL == "" || cfg.Notification.QueueName == "" {
		logger.Warn("RabbitMQ configuration missing, skipping subscriber setup")
		return nil, func() {}, nil
	}

	subscriber, err := notification.NewRabbitMQSubscriber(
		cfg.Notification.RabbitMQURL,
		cfg.Notification.QueueName,
		logger,
		handler,
		cfg.Notification.EnableConsumer, // false = consumer disabled
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RabbitMQ subscriber: %w", err)
	}

	cleanup := func() {
		logger.Info("Closing RabbitMQ subscriber")
		if err := subscriber.Close(); err != nil {
			logger.Error("Error closing RabbitMQ subscriber", zap.Error(err))
		}
	}

	return subscriber, cleanup, nil
}

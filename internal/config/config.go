package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Notification NotificationConfig `mapstructure:"notification"`
	Logging      LoggingConfig      `mapstructure:"logging"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Version     string `mapstructure:"version"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type NotificationConfig struct {
	RabbitMQURL    string   `mapstructure:"rabbitMQURL"`
	QueueName      string   `mapstructure:"queueName"`
	Subscribers    []string `mapstructure:"subscribers"`
	EnableConsumer bool     `mapstructure:"enableConsumer"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("./configs")
	viper.AutomaticEnv()

	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.sslmode", "DB_SSL_MODE")
	viper.BindEnv("notification.rabbitMQURL", "RABBITMQ_URL")
	viper.BindEnv("notification.queueName", "RABBITMQ_QUEUE_NAME")
	viper.BindEnv("notification.enableConsumer", "RABBITMQ_ENABLE_CONSUMER")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.readTimeout", "15s")
	viper.SetDefault("server.writeTimeout", "15s")
	viper.SetDefault("server.idleTimeout", "60s")
	viper.SetDefault("logging.level", "info")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	if viper.IsSet("database.port") {
		dbPortStr := viper.GetString("database.port")
		dbPort, err := strconv.Atoi(dbPortStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT value '%s': %w", dbPortStr, err)
		}
		config.Database.Port = dbPort
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Database.Host == "" {
		return fmt.Errorf("database host is not set")
	}
	if config.Database.Port == 0 {
		return fmt.Errorf("database port is not set")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is not set")
	}
	if config.Database.Name == "" {
		return fmt.Errorf("database name is not set")
	}

	if config.Notification.RabbitMQURL == "" {
		return fmt.Errorf("RabbitMQ URL is not set")
	}

	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func (c *LoggingConfig) NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(c.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

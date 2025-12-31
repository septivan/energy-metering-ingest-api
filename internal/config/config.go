package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	ServiceName            string
	ServicePort            int
	RabbitMQURL            string
	RabbitMQExchange       string
	RabbitMQRoutingKey     string
	RabbitMQMaxRetries     int
	RabbitMQRetryBaseDelay int // in milliseconds
	ServerStartTimeout     int // in seconds
	ServerStopTimeout      int // in seconds
	PublishConfirmTimeout  int // in seconds
	GinMode                string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	serviceName := getEnv("SERVICE_NAME", "energy-metering-ingest-api")
	servicePort := getEnvAsInt("SERVICE_PORT", 8080)
	rabbitMQURL := getEnv("RABBITMQ_URL", "")
	rabbitMQExchange := getEnv("RABBITMQ_EXCHANGE", "energy-metering.ingest.exchange")
	rabbitMQRoutingKey := getEnv("RABBITMQ_ROUTING_KEY", "meter.reading.ingested")
	rabbitMQMaxRetries := getEnvAsInt("RABBITMQ_MAX_RETRIES", 3)
	rabbitMQRetryBaseDelay := getEnvAsInt("RABBITMQ_RETRY_BASE_DELAY_MS", 100)
	serverStartTimeout := getEnvAsInt("SERVER_START_TIMEOUT_SEC", 15)
	serverStopTimeout := getEnvAsInt("SERVER_STOP_TIMEOUT_SEC", 15)
	publishConfirmTimeout := getEnvAsInt("PUBLISH_CONFIRM_TIMEOUT_SEC", 5)
	ginMode := getEnv("GIN_MODE", "debug")

	if rabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}

	return &Config{
		ServiceName:            serviceName,
		ServicePort:            servicePort,
		RabbitMQURL:            rabbitMQURL,
		RabbitMQExchange:       rabbitMQExchange,
		RabbitMQRoutingKey:     rabbitMQRoutingKey,
		RabbitMQMaxRetries:     rabbitMQMaxRetries,
		RabbitMQRetryBaseDelay: rabbitMQRetryBaseDelay,
		ServerStartTimeout:     serverStartTimeout,
		ServerStopTimeout:      serverStopTimeout,
		PublishConfirmTimeout:  publishConfirmTimeout,
		GinMode:                ginMode,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

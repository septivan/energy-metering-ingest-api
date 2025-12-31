package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/septivank/energy-metering-ingest-api/internal/config"
	"github.com/septivank/energy-metering-ingest-api/internal/handler"
	"github.com/septivank/energy-metering-ingest-api/internal/mq"
	"github.com/septivank/energy-metering-ingest-api/internal/service"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on configuration
	if cfg.GinMode == "release" || cfg.GinMode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	return gin.New()
}

// loadEnvFile tries to load .env file from multiple locations
// Supports both Linux (/) and Windows (\) path separators
func loadEnvFile() {
	// Possible .env file locations (in order of priority)
	envPaths := []string{
		".env",                            // Current directory
		"../../.env",                      // Project root (from cmd/server)
		filepath.Join("..", "..", ".env"), // Project root using filepath
		"/app/.env",                       // Common Kubernetes/Docker path (Linux)
		// "C:\\app\\.env",                   // Windows absolute path (if needed)
	}

	loaded := false
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err == nil {
				log.Printf("Loaded .env file from: %s", path)
				loaded = true
				break
			}
		}
	}

	if !loaded {
		log.Println("No .env file found in any location, using system environment variables")
	}
}

func main() {
	// Load .env file with flexible path handling
	loadEnvFile()

	app := fx.New(
		fx.Provide(
			config.Load,
			newLogger,
			func(cfg *config.Config, logger *zap.Logger) (*mq.Publisher, error) {
				return mq.NewPublisher(
					cfg.RabbitMQURL,
					cfg.RabbitMQExchange,
					cfg.RabbitMQMaxRetries,
					cfg.RabbitMQRetryBaseDelay,
					cfg.PublishConfirmTimeout,
					logger,
				)
			},
			func(publisher *mq.Publisher, logger *zap.Logger, cfg *config.Config) *service.IngestService {
				return service.NewIngestService(publisher, logger, cfg.RabbitMQRoutingKey)
			},
			handler.NewMeterHandler,
			handler.NewHealthHandler,
			NewRouter,
		),
		fx.Invoke(func(logger *zap.Logger, cfg *config.Config) {
			logger.Info("Configuration loaded",
				zap.String("service", cfg.ServiceName),
				zap.Int("port", cfg.ServicePort),
				zap.String("exchange", cfg.RabbitMQExchange),
			)
		}),
		fx.Invoke(startServer),
	)

	// Load config first to get timeout values
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	startCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ServerStartTimeout)*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		panic(err)
	}

	// graceful shutdown on interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ServerStopTimeout)*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		fmt.Println("error stopping app:", err)
	}
}

func startServer(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger, publisher *mq.Publisher, meterHandler *handler.MeterHandler, healthHandler *handler.HealthHandler, router *gin.Engine) {
	// register routes
	RegisterRoutes(router, meterHandler, healthHandler, logger, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServicePort),
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("starting http server", zap.Int("port", cfg.ServicePort))
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("http server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down service...")
			_ = srv.Shutdown(ctx)
			if err := publisher.Close(); err != nil {
				logger.Error("rabbitmq publisher close error", zap.Error(err))
			}
			logger.Info("service stopped")
			return nil
		},
	})
}

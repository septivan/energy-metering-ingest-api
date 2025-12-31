package main

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/septivank/energy-metering-ingest-api/internal/config"
	"github.com/septivank/energy-metering-ingest-api/internal/handler"
	"github.com/septivank/energy-metering-ingest-api/internal/middleware"
)

// RegisterRoutes registers HTTP routes on the provided Gin engine
func RegisterRoutes(r *gin.Engine, meterHandler *handler.MeterHandler, healthHandler *handler.HealthHandler, logger *zap.Logger, cfg *config.Config) {
	// Global middleware
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogger(logger))

	// Health endpoint (without service prefix for K8s probes)
	r.GET("/health", healthHandler.Check)

	// Base path with service name
	basePath := r.Group("/" + cfg.ServiceName)
	{
		// Health endpoint with service prefix
		basePath.GET("/health", healthHandler.Check)

		// API routes
		api := basePath.Group("/api/v1")
		{
			meter := api.Group("/meter")
			{
				meter.POST("/readings", meterHandler.IngestReading)
			}
		}
	}
}

package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/septivank/energy-metering-ingest-api/internal/service"
	"go.uber.org/zap"
)

// MeterHandler handles meter reading endpoints
type MeterHandler struct {
	service *service.IngestService
	logger  *zap.Logger
}

// NewMeterHandler creates a new meter handler
func NewMeterHandler(service *service.IngestService, logger *zap.Logger) *MeterHandler {
	return &MeterHandler{
		service: service,
		logger:  logger,
	}
}

// IngestReading handles POST /api/v1/meter/readings
func (h *MeterHandler) IngestReading(c *gin.Context) {
	var req service.IngestRequest

	// Bind and validate JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload",
			zap.Error(err),
			zap.String("client_ip", getClientIP(c)),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Extract client metadata
	metadata := service.ClientMetadata{
		IPAddress:     getClientIP(c),
		UserAgent:     c.GetHeader("User-Agent"),
		HasAuthHeader: c.GetHeader("Authorization") != "",
	}

	// Process reading
	if err := h.service.ProcessReading(c.Request.Context(), req, metadata); err != nil {
		h.logger.Error("Failed to process reading",
			zap.Error(err),
			zap.String("client_ip", metadata.IPAddress),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to process reading",
			"message": "Service temporarily unavailable",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "Meter reading ingested successfully",
	})
}

// getClientIP extracts the real client IP, respecting X-Forwarded-For
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

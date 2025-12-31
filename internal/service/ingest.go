package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/septivank/energy-metering-ingest-api/internal/mq"
	"github.com/septivank/energy-metering-ingest-api/tools/fingerprint"
	"go.uber.org/zap"
)

// MeterReading represents a single meter reading
type MeterReading struct {
	Date string `json:"date" binding:"required"`
	Data string `json:"data" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// IngestRequest represents the incoming request payload
type IngestRequest struct {
	PM []MeterReading `json:"PM" binding:"required,dive"`
}

// ClientMetadata represents client information
type ClientMetadata struct {
	IPAddress     string
	UserAgent     string
	HasAuthHeader bool
}

// IngestMessage represents the message to be published to RabbitMQ
type IngestMessage struct {
	RequestID         string        `json:"request_id"`
	ClientFingerprint string        `json:"client_fingerprint"`
	IPAddress         string        `json:"ip_address"`
	UserAgent         string        `json:"user_agent"`
	ReceivedAt        string        `json:"received_at"`
	Payload           IngestRequest `json:"payload"`
}

// IngestService handles meter reading ingestion
type IngestService struct {
	publisher  *mq.Publisher
	logger     *zap.Logger
	routingKey string
}

// NewIngestService creates a new ingest service
func NewIngestService(publisher *mq.Publisher, logger *zap.Logger, routingKey string) *IngestService {
	return &IngestService{
		publisher:  publisher,
		logger:     logger,
		routingKey: routingKey,
	}
}

// ProcessReading processes and publishes a meter reading
func (s *IngestService) ProcessReading(ctx context.Context, req IngestRequest, metadata ClientMetadata) error {
	// Validate PM array is not empty
	if len(req.PM) == 0 {
		return fmt.Errorf("PM array cannot be empty")
	}

	// Validate each reading has required fields
	for i, reading := range req.PM {
		if reading.Date == "" {
			return fmt.Errorf("PM[%d].date cannot be empty", i)
		}
		if reading.Data == "" {
			return fmt.Errorf("PM[%d].data cannot be empty", i)
		}
		if reading.Name == "" {
			return fmt.Errorf("PM[%d].name cannot be empty", i)
		}
	}

	// Generate request ID and fingerprint
	requestID := uuid.New().String()
	clientFingerprint := fingerprint.Generate(metadata.IPAddress, metadata.UserAgent)

	// Create message
	message := IngestMessage{
		RequestID:         requestID,
		ClientFingerprint: clientFingerprint,
		IPAddress:         metadata.IPAddress,
		UserAgent:         metadata.UserAgent,
		ReceivedAt:        time.Now().Format(time.RFC3339),
		Payload:           req,
	}

	// Publish to RabbitMQ
	if err := s.publisher.Publish(ctx, s.routingKey, message); err != nil {
		s.logger.Error("Failed to publish message",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	s.logger.Info("Meter reading ingested successfully",
		zap.String("request_id", requestID),
		zap.String("client_fingerprint", clientFingerprint),
		zap.Int("readings_count", len(req.PM)),
	)

	return nil
}

# Energy Metering Ingest API

A production-ready, stateless Go service for ingesting IoT meter readings and publishing them to RabbitMQ.

## Overview

This service is designed to be horizontally scalable and act as a lightweight ingestion layer. It:
- Receives meter readings via HTTP POST
- Performs minimal validation
- Captures client metadata
- Publishes messages to RabbitMQ with retry logic
- **Does NOT connect to any database**

## Tech Stack

- **Language:** Go 1.23
- **HTTP Framework:** Gin
- **Dependency Injection:** Uber Fx
- **Message Queue:** RabbitMQ (AMQP over TLS)
- **Logging:** Zap (structured JSON logging)
- **Configuration:** Environment variables

## Project Structure

```
energy-metering-ingest-api/
├── cmd/
│   └── server/
│       └── main.go                  # Application entry point with Fx
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration loading
│   ├── handler/
│   │   ├── meter.go                 # Meter reading HTTP handlers
│   │   └── health.go                # Health check handler
│   ├── service/
│   │   └── ingest.go                # Business logic for ingestion
│   ├── mq/
│   │   └── publisher.go             # RabbitMQ publisher with retry
│   ├── middleware/
│   │   └── middleware.go            # HTTP middleware (logging, recovery)
│   └── logging/
│       └── logger.go                # Logger initialization
├── pkg/
│   └── fingerprint/
│       └── fingerprint.go           # Client fingerprinting utility
├── go.mod
├── go.sum
└── README.md
```

## API Endpoints

### Ingest Meter Readings

**Endpoint:** `POST /api/v1/meter/readings`

**Headers:**
- `Authorization: Bearer <token>` (optional, presence is captured but not validated)
- `Content-Type: application/json`

**Request Body:**
```json
{
  "PM": [
    {
      "date": "19/12/2025 15:27:53",
      "data": "[233.336578]",
      "name": "Volts"
    },
    {
      "date": "19/12/2025 15:28:00",
      "data": "[234.123456]",
      "name": "Amps"
    }
  ]
}
```

**Response (202 Accepted):**
```json
{
  "status": "accepted",
  "message": "Meter reading ingested successfully"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid JSON or validation failure
- `503 Service Unavailable` - Failed to publish to RabbitMQ after retries

### Health Check

**Endpoint:** `GET /health`

**Response (200 OK):**
```json
{
  "status": "healthy",
  "service": "energy-metering-ingest-api"
}
```

## Validation Rules

The service performs **lightweight validation only**:
- ✅ Valid JSON structure
- ✅ `PM` field exists and is an array
- ✅ Each PM element has non-empty `date`, `data`, and `name` strings
- ❌ Does NOT validate numeric ranges
- ❌ Does NOT deduplicate readings
- ❌ Does NOT parse timestamps deeply

## Client Metadata Capture

For each request, the service captures:
- **IP Address** (respects `X-Forwarded-For`, `X-Real-IP` headers)
- **User-Agent** header
- **Authorization** header presence (boolean)
- **Request ID** (UUID v4)
- **Client Fingerprint** (SHA256 hash of IP + User-Agent)

## RabbitMQ Integration

### Message Publishing

**Exchange:** `energy-metering.ingest.exchange` (durable, topic)

**Routing Key:** `meter.reading.ingested`

**Message Format:**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_fingerprint": "a1b2c3d4e5f6...",
  "ip_address": "192.168.1.100",
  "user_agent": "IoT-Device/1.0",
  "received_at": "2025-12-29T10:30:00Z",
  "payload": {
    "PM": [...]
  }
}
```

### Reliability Features

- **Durable Exchange** - Survives broker restarts
- **Publish Confirmation** - Waits for broker acknowledgment
- **Retry Logic** - Up to 3 attempts with exponential backoff
- **Circuit Breaking** - Returns HTTP 503 if publishing fails

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SERVICE_NAME` | No | `energy-metering-ingest-api` | Service identifier |
| `SERVICE_PORT` | No | `8080` | HTTP server port |
| `RABBITMQ_URL` | **Yes** | - | AMQPS connection string (e.g., `amqps://user:pass@host/vhost`) |
| `RABBITMQ_EXCHANGE` | No | `energy-metering.ingest.exchange` | Exchange name |

### Example `.env` File

```env
SERVICE_NAME=energy-metering-ingest-api
SERVICE_PORT=8080
RABBITMQ_URL=amqps://username:password@your-cloudamqp-host.com/vhost
RABBITMQ_EXCHANGE=energy-metering.ingest.exchange
```

## Running Locally

### Prerequisites

- Go 1.23 or higher
- RabbitMQ instance (CloudAMQP recommended)

### Steps

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd energy-metering-ingest-api
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set environment variables:**
   ```bash
   export RABBITMQ_URL="amqps://your-cloudamqp-url"
   export SERVICE_PORT=8080
   ```

4. **Run the service:**
   ```bash
   go run cmd/server/main.go
   ```

5. **Test the health endpoint:**
   ```bash
   curl http://localhost:8080/health
   ```

6. **Test meter reading ingestion:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/meter/readings \
     -H "Content-Type: application/json" \
     -d '{
       "PM": [
         {
           "date": "29/12/2025 10:30:00",
           "data": "[230.5]",
           "name": "Volts"
         }
       ]
     }'
   ```

## Building for Production

### Build Binary

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server cmd/server/main.go
```

### Docker Build (Optional)

Create a `Dockerfile`:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /server .
EXPOSE 8080
CMD ["./server"]
```

Build and run:
```bash
docker build -t energy-metering-ingest-api .
docker run -p 8080:8080 \
  -e RABBITMQ_URL="amqps://..." \
  energy-metering-ingest-api
```

## Deployment on Fly.io

### 1. Install Fly CLI

```bash
# macOS/Linux
curl -L https://fly.io/install.sh | sh

# Windows
iwr https://fly.io/install.ps1 -useb | iex
```

### 2. Login to Fly.io

```bash
fly auth login
```

### 3. Create `fly.toml`

```toml
app = "energy-metering-ingest-api"
primary_region = "sin"  # Singapore - adjust to your region

[build]
  [build.args]
    GO_VERSION = "1.23"

[env]
  SERVICE_NAME = "energy-metering-ingest-api"
  SERVICE_PORT = "8080"
  RABBITMQ_EXCHANGE = "energy-metering.ingest.exchange"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [[services.tcp_checks]]
    interval = "15s"
    timeout = "2s"
    grace_period = "5s"

  [[services.http_checks]]
    interval = "10s"
    timeout = "2s"
    grace_period = "5s"
    method = "get"
    path = "/health"
```

### 4. Set Secrets

```bash
fly secrets set RABBITMQ_URL="amqps://your-cloudamqp-url"
```

### 5. Deploy

```bash
# First deployment
fly launch

# Subsequent deployments
fly deploy
```

### 6. Scale (Optional)

```bash
# Scale to 2 instances
fly scale count 2

# Scale vertically
fly scale vm shared-cpu-1x
```

### 7. Monitor

```bash
# View logs
fly logs

# Check status
fly status

# SSH into instance
fly ssh console
```

## Horizontal Scaling

This service is **fully stateless** and can be horizontally scaled:

- ✅ No database connections
- ✅ No local state
- ✅ Thread-safe RabbitMQ publisher
- ✅ Each instance operates independently

Scale freely based on throughput requirements.

## Graceful Shutdown

The service implements graceful shutdown using Uber Fx lifecycle hooks:

1. Stops accepting new HTTP requests
2. Waits for in-flight requests to complete (10s timeout)
3. Closes RabbitMQ connections cleanly
4. Flushes logs

## Logging

Structured JSON logging using Zap:

```json
{
  "level": "info",
  "ts": 1703851234.123,
  "service": "energy-metering-ingest-api",
  "msg": "Meter reading ingested successfully",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_fingerprint": "a1b2c3d4...",
  "readings_count": 2
}
```

## Performance Considerations

- **Lightweight Validation** - Minimal CPU overhead
- **Connection Pooling** - RabbitMQ channels reused
- **Non-blocking** - HTTP responses sent immediately after publish
- **Retry with Backoff** - Prevents thundering herd

## Security Notes

- ✅ Authorization header captured but **NOT validated** (delegated to downstream services)
- ✅ Client IP extracted properly from proxy headers
- ✅ TLS enforced for RabbitMQ (AMQPS)
- ⚠️ Add API gateway or authentication middleware for production

## Troubleshooting

### RabbitMQ Connection Issues

```bash
# Check connectivity
curl -I https://your-cloudamqp-host.com

# Verify credentials
# Ensure RABBITMQ_URL is properly formatted
```

### High Latency

- Check RabbitMQ broker health
- Verify network connectivity
- Monitor publish confirmation timeouts
- Scale horizontally if needed

### Memory Leaks

- Service uses structured logging (no string concatenation)
- RabbitMQ connections properly closed
- No goroutine leaks (Fx manages lifecycle)

## License

MIT

## Support

For issues or questions, please open a GitHub issue.

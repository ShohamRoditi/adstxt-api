# ads.txt Analyzer API

Production-ready RESTful API service for analyzing ads.txt files across domains.

## Features

- Single domain analysis
- Batch domain analysis (up to 50 domains)
- Pluggable cache backends (Memory, Redis, File)
- Custom rate limiting implementation (no external libraries)
- Comprehensive error handling
- Docker support with Docker Compose
- CI/CD with GitHub Actions
- Graceful shutdown
- CORS support

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker compose up -d
```

### Local Development

```bash
# Install dependencies
go mod download

# Run with memory cache
export CACHE_TYPE=memory
go run cmd/server/main.go

# Run with Redis cache
export CACHE_TYPE=redis
export REDIS_ADDR=localhost:6379
go run cmd/server/main.go
```

## API Endpoints

### Single Domain Analysis
```bash
GET /api/analyze?domain=msn.com
```

Response:
```json
{
  "domain": "msn.com",
  "total_advertisers": 189,
  "advertisers": [
    {
      "domain": "google.com",
      "count": 102
    }
  ],
  "cached": false,
  "timestamp": "2025-11-20T10:30:45Z"
}
```

### Batch Domain Analysis
```bash
POST /api/batch-analysis
Content-Type: application/json

{
  "domains": ["msn.com", "cnn.com", "vidazoo.com"]
}
```

Response:
```json
{
  "results": [
    {
      "domain": "msn.com",
      "total_advertisers": 189,
      "advertisers": [...],
      "cached": false,
      "timestamp": "2025-11-20T10:30:45Z"
    }
  ],
  "errors": {
    "invalid-domain.com": "failed to fetch ads.txt"
  }
}
```

### Health Check
```bash
GET /health
```

### Metrics
```bash
GET /metrics
```

Response:
```json
{
  "requests_total": 1523,
  "cache_hits": 892,
  "cache_misses": 631,
  "errors_total": 12
}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Server port |
| CACHE_TYPE | memory | Cache backend: memory, redis, file |
| CACHE_TTL | 1h | Cache time-to-live |
| RATE_LIMIT_PER_SECOND | 10 | Rate limit per client |
| REDIS_ADDR | localhost:6379 | Redis address |
| REDIS_PASSWORD | "" | Redis password |
| REDIS_DB | 0 | Redis database |
| FILE_STORAGE_PATH | ./cache | File cache path |
| REQUEST_TIMEOUT | 10s | HTTP request timeout |

## Testing

```bash
# Run all tests
go test ./... -v

# Run with race detector and coverage
go test -race -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -html=coverage.out

# Or use Makefile
make test
```

## Architecture

### Rate Limiter
Custom implementation using token bucket algorithm with per-client tracking. Automatically cleans up inactive clients every minute.

### Cache System
Abstract cache interface with three implementations:
- **Memory**: In-memory cache with TTL and automatic cleanup
- **Redis**: Distributed cache using Redis
- **File**: Filesystem-based cache for persistence

### Concurrent Processing
Batch requests process domains concurrently using goroutines with proper synchronization.

## Make Commands

```bash
make build          # Build the server binary
make test           # Run tests with coverage
make run            # Run the server locally
make docker-build   # Build Docker image
make docker-up      # Start services with Docker Compose
make docker-down    # Stop services
make docker-logs    # View service logs
make clean          # Clean build artifacts and stop containers
make lint           # Run linter
make deps           # Download and tidy dependencies
```

## Production Considerations

- Graceful shutdown with 30s timeout
- Connection timeouts and limits
- Comprehensive logging
- Error handling with proper HTTP status codes
- CORS support for web clients
- Rate limiting per client IP
- Cache to reduce external requests
- Concurrent batch processing

## License

MIT
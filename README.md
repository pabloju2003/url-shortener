# url-shortener

A high-performance URL shortener built in Go, designed to handle high concurrency using goroutines and channels.

## Features

- **URL shortening** — generates unique 6-character codes using crypto/rand
- **Fast redirects** — Redis cache layer for sub-millisecond lookups
- **Async click tracking** — goroutine-based click recording that never blocks redirects
- **Click statistics** — per-URL analytics with country and device breakdown
- **Rate limiting** — token bucket implementation using Go channels (no external libraries)
- **Database migrations** — automatic schema management on startup

## Benchmark

Tested locally with Docker (50 concurrent goroutines, 1000 requests):

| Metric | Result |
|--------|--------|
| Requests/sec | 724 |
| Avg latency | 67ms |
| Min latency | 36ms |
| Max latency | 139ms |
| Success rate | 100% |

## Tech stack

- **Go 1.24** with Gin framework
- **PostgreSQL 16** — persistent storage
- **Redis 7** — URL cache with 24h TTL
- **Docker** — containerized setup
- **golang-migrate** — database migrations

## API

### POST /shorten
Creates a short URL.

Request:
```json
{
  "url": "https://example.com"
}
```

Response (201):
```json
{
  "code": "McoyYm",
  "short_url": "http://localhost:8080/McoyYm",
  "original_url": "https://example.com"
}
```

### GET /:code
Redirects to the original URL (301). Records the click asynchronously.

### GET /stats/:code
Returns click analytics for a short URL.

Response (200):
```json
{
  "url": { "id": 1, "code": "McoyYm", "original_url": "https://example.com" },
  "total_clicks": 1042,
  "top_countries": { "": 1042 },
  "top_devices": { "desktop": 800, "mobile": 242 }
}
```

### GET /health
Returns service status.

## How concurrency works

Two Go concurrency patterns are used in this project:

**Async click tracking**: when a redirect is served, a goroutine is launched to record the click without blocking the HTTP response. The user gets the redirect in microseconds while tracking happens in the background.

**Token bucket rate limiter**: built from scratch using a buffered channel as the token store and a goroutine that refills it at a fixed rate. No external libraries — just Go primitives.

## Getting started

### Prerequisites
- Docker and Docker Compose

### Run

```bash
git clone https://github.com/pabloju2003/url-shortener.git
cd url-shortener
cp .env.example .env
docker-compose up --build
```

The server starts at `http://localhost:8080`. Migrations run automatically.

### Run tests

```bash
go test ./...
```

### Run benchmark

```bash
go run cmd/benchmark/main.go <code>
```

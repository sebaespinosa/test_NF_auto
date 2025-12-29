# Irrigation Analytics Service

A REST API service for managing and analyzing irrigation data for agricultural platforms. Built with Go, Gin, GORM, and PostgreSQL.

## Features

- **Irrigation Analytics Endpoint**: Provides comprehensive analytics with year-over-year comparisons
- **Time-Series Aggregation**: Daily, weekly, and monthly data aggregation
- **Sector Breakdown**: Per-sector performance metrics
- **Database Optimization**: Efficient queries with proper indexing for large datasets
- **Observability**: Structured logging and in-memory metrics
- **HTTPS Support**: Production-ready with TLS support

## Architecture

The service follows clean architecture principles:

```
├── cmd/
│   ├── server/          # Application entry point
│   └── seed/            # Database seeding utility
├── internal/
│   ├── controller/      # HTTP handlers (Gin)
│   ├── service/         # Business logic
│   ├── repository/      # Data access layer (GORM)
│   └── model/          # Database models
├── pkg/
│   ├── logger/         # Structured logging
│   └── metrics/        # In-memory metrics tracking
└── docker-compose.yml  # Development environment
```

## Database Schema

### Tables

**farms**
- `id` (primary key)
- `name`
- `created_at`, `updated_at`

**irrigation_sectors**
- `id` (primary key)
- `farm_id` (foreign key, indexed)
- `name`
- `created_at`, `updated_at`

**irrigation_data**
- `id` (primary key)
- `farm_id` (foreign key, composite index with start_time)
- `irrigation_sector_id` (foreign key, composite index with start_time)
- `start_time` (indexed)
- `end_time`
- `nominal_amount` (numeric, mm)
- `real_amount` (numeric, mm)
- `created_at`, `updated_at`

### Indexes

For optimal query performance, the following indexes are created:

1. **idx_farm_time**: Composite index on `(farm_id, start_time)` - Optimizes time-range queries per farm
2. **idx_sector_time**: Composite index on `(irrigation_sector_id, start_time)` - Optimizes queries filtered by sector

These indexes ensure efficient:
- Time-range filtering (WHERE start_time >= ? AND start_time < ?)
- Farm-based filtering (WHERE farm_id = ?)
- Sector-based filtering (WHERE irrigation_sector_id = ?)
- Aggregation queries (GROUP BY with date truncation)

## API Documentation

### Analytics Endpoint

**GET** `/v1/farms/{farm_id}/irrigation/analytics`

Retrieves comprehensive irrigation analytics including historical comparisons.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `start_date` | ISO 8601 string | No | 30 days ago | Start of period (e.g., "2024-01-01" or "2024-01-01T00:00:00Z") |
| `end_date` | ISO 8601 string | No | Current time | End of period |
| `sector_id` | integer | No | - | Filter by specific irrigation sector |
| `aggregation` | string | No | "daily" | Aggregation level: "daily", "weekly", or "monthly" |

#### Response Format

```json
{
  "farm_id": 123,
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "aggregation": "daily",
  "metrics": {
    "total_irrigation_volume_mm": 450.5,
    "total_irrigation_events": 120,
    "average_efficiency": 0.85,
    "efficiency_range": {
      "min": 0.72,
      "max": 0.98
    },
    "same_period_-1": {
      "total_irrigation_volume_mm": 420.3,
      "total_irrigation_events": 115,
      "average_efficiency": 0.82,
      "efficiency_range": {
        "min": 0.70,
        "max": 0.95
      }
    },
    "same_period_-2": {
      "total_irrigation_volume_mm": 480.1,
      "total_irrigation_events": 125,
      "average_efficiency": 0.88,
      "efficiency_range": {
        "min": 0.75,
        "max": 0.99
      }
    },
    "period_comparison": {
      "vs_same_period_-1": {
        "volume_change_percent": 7.2,
        "events_change_percent": 4.3,
        "efficiency_change_percent": 3.7
      },
      "vs_same_period_-2": {
        "volume_change_percent": -6.2,
        "events_change_percent": -4.0,
        "efficiency_change_percent": -3.4
      }
    }
  },
  "time_series": [
    {
      "date": "2024-01-01",
      "nominal_amount_mm": 12.5,
      "real_amount_mm": 10.8,
      "efficiency": 0.864,
      "event_count": 3
    }
  ],
  "sector_breakdown": [
    {
      "sector_id": 1,
      "sector_name": "Sector A",
      "total_volume_mm": 150.2,
      "average_efficiency": 0.88
    }
  ]
}
```

#### Efficiency Calculation

Efficiency is calculated as:
```
efficiency = real_amount / nominal_amount
```

Edge cases:
- When `nominal_amount` is 0, efficiency is excluded from calculations
- NULL values are handled gracefully

#### Year-over-Year Comparisons

The service automatically calculates metrics for:
- **same_period_-1**: Same date range from 1 year ago
- **same_period_-2**: Same date range from 2 years ago

Example: For period `2024-01-01` to `2024-01-31`:
- **same_period_-1**: `2023-01-01` to `2023-01-31`
- **same_period_-2**: `2022-01-01` to `2022-01-31`

Percentage changes are calculated as:
```
change_percent = ((current - previous) / previous) * 100
```

If historical data doesn't exist, those fields are omitted from the response.

### Health Check

**GET** `/health`

Returns service health status.

```json
{
  "status": "healthy"
}
```

### Metrics

**GET** `/metrics`

Returns service metrics.

```json
{
  "total_requests": 1234,
  "total_errors": 5,
  "average_latency": "45ms",
  "requests_by_path": {
    "/v1/farms/:farm_id/irrigation/analytics": 1234
  }
}
```

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)

### Running with Docker Compose

1. **Start the services:**
```bash
docker-compose up -d
```

This will start:
- PostgreSQL database on port 5432
- API server on HTTPS port 8443

2. **Seed the database:**
```bash
docker-compose exec app go run cmd/seed/main.go
```

3. **Test the API:**
```bash
# Get analytics for farm 1 (skip certificate verification for self-signed cert)
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-01-31&aggregation=daily"

# Get weekly analytics for a specific sector
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?sector_id=1&aggregation=weekly"

# Get monthly analytics
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?aggregation=monthly"
```

### Local Development

1. **Install dependencies:**
```bash
go mod download
```

2. **Start PostgreSQL:**
```bash
docker-compose up -d postgres
```

3. **Run the server:**
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=irrigation_analytics
export PORT=8443

go run cmd/server/main.go
```

4. **Seed the database:**
```bash
go run cmd/seed/main.go
```

### Testing

Run unit tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run specific test:
```bash
go test ./internal/service/... -v
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | PostgreSQL database name | `irrigation_analytics` |
| `PORT` | Server port | `8443` |
| `TLS_CERT` | Path to TLS certificate | - |
| `TLS_KEY` | Path to TLS private key | - |

## Database Optimization

The service implements several optimizations for handling large datasets:

### Query Optimizations

1. **Database-level Aggregations**: All aggregations (SUM, AVG, MIN, MAX, COUNT) are performed at the database level using efficient SQL queries rather than loading data into application memory.

2. **Composite Indexes**: 
   - `idx_farm_time` on `(farm_id, start_time)` for efficient farm-specific queries
   - `idx_sector_time` on `(irrigation_sector_id, start_time)` for sector-specific queries

3. **Single Query Pattern**: Uses single queries with JOINs instead of N+1 query patterns for sector breakdown.

4. **Selective Field Selection**: Only retrieves necessary fields using `SELECT` statements.

### Performance Considerations

- Time-range queries use index-friendly comparisons: `start_time >= ? AND start_time < ?`
- Date truncation happens at database level using PostgreSQL's `DATE_TRUNC` function
- Efficiency calculations use CASE statements to handle edge cases at database level
- COALESCE used to handle NULL values in aggregations

### Testing Query Performance

Use PostgreSQL's `EXPLAIN ANALYZE` to verify query performance:

```sql
EXPLAIN ANALYZE
SELECT 
    COALESCE(SUM(real_amount), 0) as total_volume,
    COUNT(*) as total_events,
    COALESCE(AVG(CASE WHEN nominal_amount > 0 THEN real_amount / nominal_amount ELSE NULL END), 0) as average_efficiency
FROM irrigation_data
WHERE farm_id = 1 AND start_time >= '2024-01-01' AND start_time < '2024-02-01';
```

## Observability

### Structured Logging

The service uses structured logging with the following information:

- Request parameters (farm_id, dates, aggregation)
- Query timing
- Result counts
- Errors with context

Example log output:
```
INFO Processing analytics request farm_id=1 start_date=2024-01-01T00:00:00Z end_date=2024-01-31T00:00:00Z aggregation=daily
INFO Analytics request completed farm_id=1 latency_ms=45 result_count=31
```

### Metrics

In-memory metrics track:
- Total request count
- Error count
- Average latency
- Requests by endpoint

Access metrics at `/metrics` endpoint.

## Error Handling

The service handles various error scenarios gracefully:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| Invalid farm_id | 400 | Non-numeric or invalid farm ID |
| Invalid dates | 400 | Malformed date strings |
| Invalid date range | 400 | end_date before start_date |
| Invalid sector_id | 400 | Non-numeric sector ID |
| Invalid aggregation | 400 | Aggregation not in [daily, weekly, monthly] |
| Database error | 500 | Database connection or query failure |

All errors return JSON:
```json
{
  "error": "Description of the error"
}
```

## HTTPS/TLS Configuration

### Development

The Docker container automatically generates a self-signed certificate for development. Use `-k` flag with curl to skip verification:

```bash
curl -k "https://localhost:8443/..."
```

### Production

Set environment variables to use your own certificates:

```bash
export TLS_CERT=/path/to/certificate.pem
export TLS_KEY=/path/to/private-key.pem
```

## License

MIT

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

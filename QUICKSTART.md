# Quick Start Guide - Irrigation Analytics Service

This guide will help you quickly get the Irrigation Analytics Service up and running.

## Prerequisites

- Docker and Docker Compose installed
- OR Go 1.21+ for local development

## Option 1: Quick Start with Docker Compose (Recommended)

### 1. Clone the repository

```bash
git clone https://github.com/sebaespinosa/test_NF_auto.git
cd test_NF_auto
```

### 2. Start the services

```bash
docker compose up -d
```

This will start:
- PostgreSQL database on port 5432
- API server on HTTPS port 8443

Wait a few seconds for the database to initialize.

### 3. Seed the database with sample data

```bash
# Wait for the app to be ready
sleep 10

# Run the seed script inside the container
docker compose exec app go run cmd/seed/main.go
```

This will create:
- 3 farms
- 12 irrigation sectors (4 per farm)
- ~4,000+ irrigation events spanning 2022-2024

### 4. Test the API

```bash
# Get daily analytics for farm 1 in January 2024
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-01-31&aggregation=daily"

# Get weekly analytics for 2024
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-12-31&aggregation=weekly"

# Get monthly analytics for a specific sector
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?sector_id=1&aggregation=monthly"

# Check service health
curl -k "https://localhost:8443/health"

# View metrics
curl -k "https://localhost:8443/metrics"
```

**Note**: The `-k` flag is used to skip certificate verification since we're using a self-signed certificate in development.

## Option 2: Local Development (Without Docker)

### 1. Start PostgreSQL

```bash
# Using Docker for PostgreSQL only
docker run -d \
  --name postgres-irrigation \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=irrigation_analytics \
  -p 5432:5432 \
  postgres:15-alpine
```

### 2. Set environment variables

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=irrigation_analytics
export PORT=8080
```

### 3. Install dependencies and run

```bash
go mod download
go run cmd/server/main.go
```

The server will start on HTTP port 8080 (no TLS in dev mode without certificates).

### 4. Seed the database

In another terminal:

```bash
go run cmd/seed/main.go
```

### 5. Test the API

```bash
# Note: Using HTTP (not HTTPS) and port 8080
curl "http://localhost:8080/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-01-31"
```

## Understanding the Response

The API returns comprehensive analytics:

```json
{
  "farm_id": 1,
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "aggregation": "daily",
  "metrics": {
    "total_irrigation_volume_mm": 450.5,      // Total water volume in mm
    "total_irrigation_events": 120,           // Number of irrigation events
    "average_efficiency": 0.85,               // Overall efficiency (real/nominal)
    "efficiency_range": {
      "min": 0.72,                           // Lowest efficiency
      "max": 0.98                            // Highest efficiency
    },
    "same_period_-1": {                      // Data from 1 year ago
      "total_irrigation_volume_mm": 420.3,
      "total_irrigation_events": 115,
      "average_efficiency": 0.82,
      // ...
    },
    "same_period_-2": {                      // Data from 2 years ago
      // ...
    },
    "period_comparison": {
      "vs_same_period_-1": {
        "volume_change_percent": 7.2,        // % change vs last year
        "events_change_percent": 4.3,
        "efficiency_change_percent": 3.7
      },
      "vs_same_period_-2": {
        // % change vs 2 years ago
      }
    }
  },
  "time_series": [                           // Daily/weekly/monthly data points
    {
      "date": "2024-01-01",
      "nominal_amount_mm": 12.5,
      "real_amount_mm": 10.8,
      "efficiency": 0.864,
      "event_count": 3
    }
    // ... more entries
  ],
  "sector_breakdown": [                      // Per-sector statistics
    {
      "sector_id": 1,
      "sector_name": "Sector A",
      "total_volume_mm": 150.2,
      "average_efficiency": 0.88
    }
    // ... more sectors
  ]
}
```

## Common Use Cases

### 1. Dashboard Overview (Last 30 days)

```bash
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics"
```

Default shows last 30 days with daily aggregation.

### 2. Year-over-Year Comparison

```bash
# Compare January 2024 vs 2023 vs 2022
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-01-31"
```

The response includes `same_period_-1` and `same_period_-2` for comparison.

### 3. Weekly Trends

```bash
# Get weekly aggregation for Q1 2024
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-03-31&aggregation=weekly"
```

### 4. Sector-Specific Analysis

```bash
# Analyze specific irrigation sector
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?sector_id=1&start_date=2024-01-01&end_date=2024-12-31&aggregation=monthly"
```

### 5. Full Year Summary

```bash
# Monthly summary for entire year
curl -k "https://localhost:8443/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-12-31&aggregation=monthly"
```

## Troubleshooting

### Database Connection Issues

Check if PostgreSQL is running:
```bash
docker compose ps
```

Check logs:
```bash
docker compose logs postgres
docker compose logs app
```

### No Data Returned

Make sure you've seeded the database:
```bash
docker compose exec app go run cmd/seed/main.go
```

### Port Already in Use

If port 8443 or 5432 is already in use, modify `docker-compose.yml`:
```yaml
ports:
  - "8444:8443"  # Use different host port
```

### TLS Certificate Errors

In development, use `-k` flag with curl to skip certificate verification. For production, provide proper certificates via `TLS_CERT` and `TLS_KEY` environment variables.

## Next Steps

- Review the [README.md](README.md) for detailed API documentation
- Explore the database schema in the models
- Check out the test files for examples
- Modify the seed script to generate custom test data

## Stopping the Service

```bash
docker compose down

# To also remove volumes (database data)
docker compose down -v
```

## Production Deployment

For production:

1. **Use real TLS certificates**:
   ```bash
   export TLS_CERT=/path/to/cert.pem
   export TLS_KEY=/path/to/key.pem
   ```

2. **Use secure database credentials**:
   Update `DB_PASSWORD` environment variable

3. **Configure proper database backups**

4. **Set up monitoring and alerting**

5. **Review and adjust database indexes** based on query patterns

6. **Consider connection pooling** for high-traffic scenarios

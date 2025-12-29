# Database Schema and Migrations

This document describes the database schema, indexes, and migration strategy for the Irrigation Analytics Service.

## Overview

The service uses PostgreSQL as the database and GORM for ORM and auto-migrations. The schema is designed to efficiently handle millions of irrigation events with optimized indexes for time-range queries.

## Schema

### Tables

#### 1. farms

Stores farm information.

```sql
CREATE TABLE farms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);
```

**Fields:**
- `id`: Unique identifier (auto-increment)
- `name`: Farm name
- `created_at`, `updated_at`: Timestamps for record tracking

---

#### 2. irrigation_sectors

Stores irrigation sectors within farms.

```sql
CREATE TABLE irrigation_sectors (
    id SERIAL PRIMARY KEY,
    farm_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (farm_id) REFERENCES farms(id)
);

CREATE INDEX idx_irrigation_sectors_farm_id ON irrigation_sectors(farm_id);
```

**Fields:**
- `id`: Unique identifier (auto-increment)
- `farm_id`: Foreign key to farms table
- `name`: Sector name
- `created_at`, `updated_at`: Timestamps

**Indexes:**
- `idx_irrigation_sectors_farm_id`: Single-column index on farm_id for efficient lookups

---

#### 3. irrigation_data

Stores individual irrigation events with nominal and real amounts.

```sql
CREATE TABLE irrigation_data (
    id SERIAL PRIMARY KEY,
    farm_id INTEGER NOT NULL,
    irrigation_sector_id INTEGER NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    nominal_amount NUMERIC(10,2),  -- in mm
    real_amount NUMERIC(10,2),     -- in mm
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (farm_id) REFERENCES farms(id),
    FOREIGN KEY (irrigation_sector_id) REFERENCES irrigation_sectors(id)
);

-- Composite index for farm + time range queries
CREATE INDEX idx_irrigation_data_farm_time ON irrigation_data(farm_id, start_time);

-- Composite index for sector + time range queries
CREATE INDEX idx_irrigation_data_sector_time ON irrigation_data(irrigation_sector_id, start_time);
```

**Fields:**
- `id`: Unique identifier (auto-increment)
- `farm_id`: Foreign key to farms table
- `irrigation_sector_id`: Foreign key to irrigation_sectors table
- `start_time`: When irrigation started (indexed)
- `end_time`: When irrigation ended
- `nominal_amount`: Planned irrigation amount in millimeters (precision: 10,2)
- `real_amount`: Actual irrigation amount in millimeters (precision: 10,2)
- `created_at`, `updated_at`: Timestamps

**Indexes:**
- `idx_irrigation_data_farm_time`: Composite index on (farm_id, start_time)
  - Optimizes queries filtering by farm and time range
  - Used for analytics queries that fetch data for a specific farm within a date range
- `idx_irrigation_data_sector_time`: Composite index on (irrigation_sector_id, start_time)
  - Optimizes queries filtering by sector and time range
  - Used when requesting analytics for a specific sector

## Index Strategy

### Why Composite Indexes?

The application's primary query pattern is:
```sql
SELECT ... FROM irrigation_data
WHERE farm_id = ? AND start_time >= ? AND start_time < ?
[AND irrigation_sector_id = ?]
```

Composite indexes on `(farm_id, start_time)` and `(irrigation_sector_id, start_time)` provide:

1. **Efficient filtering**: PostgreSQL can use the index to quickly locate matching rows
2. **Range scans**: The B-tree index structure enables efficient range queries on start_time
3. **Reduced I/O**: Fewer disk reads compared to separate indexes

### Index Performance

With the composite indexes:
- Farm-based queries: `O(log n)` lookup + sequential scan of matching time range
- Sector-based queries: `O(log n)` lookup + sequential scan of matching time range
- Without indexes: `O(n)` full table scan

For a table with 10 million rows:
- **With index**: ~1000x faster (milliseconds vs seconds)
- **Without index**: Full table scan required

## Query Patterns and Optimization

### 1. Analytics for a Farm

```sql
SELECT 
    COALESCE(SUM(real_amount), 0) as total_volume,
    COUNT(*) as total_events,
    COALESCE(AVG(CASE WHEN nominal_amount > 0 THEN real_amount / nominal_amount ELSE NULL END), 0) as average_efficiency
FROM irrigation_data
WHERE farm_id = $1 AND start_time >= $2 AND start_time < $3;
```

**Index used**: `idx_irrigation_data_farm_time`

### 2. Time-Series Aggregation

```sql
SELECT 
    TO_CHAR(DATE_TRUNC('day', start_time), 'YYYY-MM-DD') as date,
    COALESCE(SUM(nominal_amount), 0) as nominal_amount,
    COALESCE(SUM(real_amount), 0) as real_amount,
    COUNT(*) as event_count
FROM irrigation_data
WHERE farm_id = $1 AND start_time >= $2 AND start_time < $3
GROUP BY date
ORDER BY date;
```

**Index used**: `idx_irrigation_data_farm_time`

**Optimization**: 
- Date truncation happens at database level
- Aggregation uses efficient hash aggregation
- Result set is small (one row per day/week/month)

### 3. Sector Breakdown

```sql
SELECT 
    irrigation_data.irrigation_sector_id as sector_id,
    irrigation_sectors.name as sector_name,
    COALESCE(SUM(irrigation_data.real_amount), 0) as total_volume,
    COALESCE(AVG(CASE WHEN irrigation_data.nominal_amount > 0 THEN irrigation_data.real_amount / irrigation_data.nominal_amount ELSE NULL END), 0) as average_efficiency
FROM irrigation_data
LEFT JOIN irrigation_sectors ON irrigation_sectors.id = irrigation_data.irrigation_sector_id
WHERE irrigation_data.farm_id = $1 AND irrigation_data.start_time >= $2 AND irrigation_data.start_time < $3
GROUP BY irrigation_data.irrigation_sector_id, irrigation_sectors.name
ORDER BY irrigation_data.irrigation_sector_id;
```

**Index used**: `idx_irrigation_data_farm_time` for filtering, `idx_irrigation_sectors_farm_id` for join

## Testing Index Performance

Use `EXPLAIN ANALYZE` to verify index usage:

```sql
EXPLAIN ANALYZE
SELECT 
    COALESCE(SUM(real_amount), 0) as total_volume,
    COUNT(*) as total_events
FROM irrigation_data
WHERE farm_id = 1 AND start_time >= '2024-01-01' AND start_time < '2024-02-01';
```

Expected output should show:
```
Index Scan using idx_irrigation_data_farm_time on irrigation_data
  Index Cond: ((farm_id = 1) AND (start_time >= '2024-01-01') AND (start_time < '2024-02-01'))
```

## Migration Strategy

### Auto-Migration (Development)

The application uses GORM's AutoMigrate for development:

```go
db.AutoMigrate(&model.Farm{}, &model.IrrigationSector{}, &model.IrrigationData{})
```

This automatically:
- Creates tables if they don't exist
- Adds missing columns
- Creates indexes defined in struct tags

### Manual Migration (Production)

For production, we recommend using migration files:

#### Step 1: Create Initial Schema

```sql
-- migrations/001_initial_schema.up.sql
BEGIN;

CREATE TABLE farms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE irrigation_sectors (
    id SERIAL PRIMARY KEY,
    farm_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (farm_id) REFERENCES farms(id)
);

CREATE INDEX idx_irrigation_sectors_farm_id ON irrigation_sectors(farm_id);

CREATE TABLE irrigation_data (
    id SERIAL PRIMARY KEY,
    farm_id INTEGER NOT NULL,
    irrigation_sector_id INTEGER NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    nominal_amount NUMERIC(10,2),
    real_amount NUMERIC(10,2),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (farm_id) REFERENCES farms(id),
    FOREIGN KEY (irrigation_sector_id) REFERENCES irrigation_sectors(id)
);

CREATE INDEX idx_irrigation_data_farm_time ON irrigation_data(farm_id, start_time);
CREATE INDEX idx_irrigation_data_sector_time ON irrigation_data(irrigation_sector_id, start_time);

COMMIT;
```

#### Step 2: Create Rollback

```sql
-- migrations/001_initial_schema.down.sql
BEGIN;

DROP TABLE IF EXISTS irrigation_data;
DROP TABLE IF EXISTS irrigation_sectors;
DROP TABLE IF EXISTS farms;

COMMIT;
```

## Performance Considerations

### Write Performance

- Composite indexes add overhead to INSERT operations
- With 2 indexes on irrigation_data, each INSERT updates 2 index structures
- Trade-off: ~10-20% slower writes for 100-1000x faster reads

### Read Performance

- Queries using indexes: O(log n) + range scan
- Aggregations happen at database level (fast)
- Time-series queries return small result sets (< 1000 rows typically)

### Storage

- Each index consumes additional disk space
- Estimate: ~10-15% of table size per index
- For 10M rows: ~500MB table + ~150MB indexes

## Monitoring

### Query Performance

Check slow queries:
```sql
-- Enable slow query logging in postgresql.conf
log_min_duration_statement = 1000  # Log queries > 1 second

-- View current queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 seconds';
```

### Index Usage

Check if indexes are being used:
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes 
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

Low `idx_scan` values indicate unused indexes.

## Future Optimizations

Consider these optimizations as data grows:

1. **Partitioning**: Partition `irrigation_data` by date (monthly or yearly)
2. **Materialized Views**: Pre-aggregate monthly/yearly statistics
3. **Archiving**: Move old data to separate archive tables
4. **Additional Indexes**: Add partial indexes for frequent query patterns
5. **Query Caching**: Implement application-level caching for common queries

## References

- [PostgreSQL Index Documentation](https://www.postgresql.org/docs/current/indexes.html)
- [GORM Migrations](https://gorm.io/docs/migration.html)
- [PostgreSQL Performance Optimization](https://www.postgresql.org/docs/current/performance-tips.html)

# Location Service

Read-only Indonesian administrative location service for provinces, regencies, districts, and villages. Data is imported from `wilayah.sql`, normalized into PostgreSQL, cached in Redis, and exposed through a small HTTP API with starter-kit style responses.

[Try the Frontend](https://location-service-do.vercel.app/) · [Health Check](https://location-service-y7si.onrender.com/healthz) · [Postman Collection](postman/location-service.postman_collection.json)

## Overview

`location-service` exists so other projects do not need to depend on third-party location APIs at runtime. It becomes the internal source of truth for location data and keeps import, normalization, and cache behavior in one small service.

Runtime read flow:

```text
Client -> HTTP API -> Redis -> PostgreSQL -> Response
```

If Redis is unavailable, the API logs the error and continues with PostgreSQL only.

## Features

- Province, regency/city, district, and village lookup APIs.
- Search API across all location levels.
- PostgreSQL normalized tables for stable source-of-truth data.
- Redis cache before database reads, with default TTL of six months.
- One-time automatic seed on first startup when `raw_locations` is empty.
- Starter-kit response envelope for success and error responses.
- Full code and short code response modes for compatibility with existing clients.
- Separate backend and frontend Dockerfiles.
- Static frontend console for browsing and testing location data.

## Stack

| Area | Technology |
|------|------------|
| Backend | Go 1.26, standard library HTTP server |
| Database | PostgreSQL |
| Cache | Redis |
| Driver | `github.com/lib/pq` |
| Redis Client | `github.com/redis/go-redis/v9` |
| Frontend | Static HTML, CSS, JavaScript |
| Runtime | Docker, Docker Compose |

## Project Structure

```text
.
├── cmd/importer/                 # Bulk import command logic
├── data/                         # Bundled wilayah.sql seed file
├── frontend/                     # Static frontend app and frontend Dockerfile
├── infrastructure/database/      # PostgreSQL and Redis connections
├── internal/bootstrap/           # Startup seed orchestration
├── internal/cache/location/      # Redis cache helpers and keys
├── internal/domain/location/     # Location entity and import stats
├── internal/handlers/http/       # HTTP handlers
├── internal/interfaces/location/ # Service and repository contracts
├── internal/repositories/        # SQL queries
├── internal/router/              # Route registration
├── internal/services/            # Validation and use-case logic
├── middlewares/                  # HTTP middleware
├── migrations/                   # SQL schema
├── pkg/messages/                 # Shared response messages
├── pkg/response/                 # Starter-kit style response envelope
├── postman/                      # Postman collection
├── utils/                        # Env helpers
└── main.go                       # Command entrypoint
```

## Data Model

Source code format:

```text
11                 province
11.01              regency/city
11.01.01           district
11.01.01.2001      village
```

Normalized tables:

| Table | Rows |
|-------|------|
| `raw_locations` | 91,599 |
| `provinces` | 38 |
| `regencies` | 514 |
| `districts` | 7,285 |
| `villages` | 83,762 |

## Configuration

Create a local environment file:

```bash
cp .env.example .env
```

Main variables:

```env
APP_ENV=development
PORT=8080
PATH_MIGRATE=migrations/000001_init.sql
CORS_ALLOWED_ORIGINS=*
COMMAND=serve

DATABASE_URL=postgres://location:location@localhost:5438/location?sslmode=disable

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
LOCATION_CACHE_TTL=4320h
```

If `DATABASE_URL` is empty, the service builds a PostgreSQL DSN from:

```env
DB_HOST=localhost
DB_PORT=5438
DB_USERNAME=location
DB_PASS=location
DB_NAME=location
DB_SSLMODE=disable
```

Seed and import variables:

```env
AUTO_SEED=true
AUTO_SEED_REQUIRED=false
SEED_FILE=data/wilayah.sql

RUN_MIGRATION=false
RUN_IMPORT=false
IMPORT_FILE=/app/data/wilayah.sql
```

`AUTO_SEED=true` runs during `serve`. It checks `raw_locations`; if the table is empty, it imports `SEED_FILE` in one transaction using PostgreSQL `COPY`. Startup skips seeding when data already exists.

## Quick Start

Start PostgreSQL and Redis:

```bash
docker compose up -d postgres redis
```

Run migration:

```bash
go run . migrate
```

Import data:

```bash
go run . import -file data/wilayah.sql
```

Start the API:

```bash
go run . serve
```

API:

```text
http://localhost:8080
```

Frontend, without Docker:

```bash
cd frontend
python3 -m http.server 5173
```

Then open:

```text
http://localhost:5173
```

## Commands

```bash
go run . migrate
```

Creates or updates database schema.

```bash
go run . import -file data/wilayah.sql
```

Imports `wilayah.sql` into `raw_locations`, then bulk-loads normalized tables. Import truncates existing location tables by default.

```bash
go run . serve
```

Runs migration, runs first-start seed if needed, connects to Redis when available, and starts the read-only HTTP API.

## API Response Format

Success:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 200,
  "status": true,
  "message": "Success",
  "data": []
}
```

Error:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 400,
  "status": false,
  "message": "Bad Request",
  "error": {
    "code": 400,
    "message": "province_code is required"
  }
}
```

## API Endpoints

### Health

```text
GET /healthz
```

### Provinces

```text
GET /api/locations/provinces
```

Example item:

```json
{
  "code": "11",
  "full_code": "11",
  "name": "Aceh",
  "level": "province"
}
```

### Regencies

```text
GET /api/locations/regencies?province_code=11
GET /api/locations/regencies?province_code=11&code_format=short
```

Full code item:

```json
{
  "code": "11.01",
  "full_code": "11.01",
  "name": "Kabupaten Aceh Selatan",
  "level": "regency",
  "parent_code": "11"
}
```

Short code item:

```json
{
  "code": "01",
  "full_code": "11.01",
  "name": "Kabupaten Aceh Selatan",
  "level": "regency",
  "parent_code": "11"
}
```

### Districts

```text
GET /api/locations/districts?regency_code=11.01
GET /api/locations/districts?province_code=11&regency_code=01&code_format=short
```

Example short code item:

```json
{
  "code": "01",
  "full_code": "11.01.01",
  "name": "Bakongan",
  "level": "district",
  "parent_code": "11.01"
}
```

### Villages

```text
GET /api/locations/villages?district_code=11.01.01
GET /api/locations/villages?province_code=11&regency_code=01&district_code=01&code_format=short
```

Example short code item:

```json
{
  "code": "2001",
  "full_code": "11.01.01.2001",
  "name": "Keude Bakongan",
  "level": "village",
  "parent_code": "11.01.01"
}
```

### Search

```text
GET /api/locations/search?q=aceh
GET /api/locations/search?q=aceh&limit=5
```

Rules:

- `q` is required.
- `limit` defaults to `50`.
- `limit` must be between `1` and `500`.

## Code Format

Default API responses use full administrative codes:

```text
province: 11
regency: 11.01
district: 11.01.01
village: 11.01.01.2001
```

For clients that need short child codes, pass `code_format=short`:

```text
regency: 01
district: 01
village: 2001
```

`full_code` is always included, so clients can migrate from short code to full code gradually.

## Redis Cache

Cache keys are scoped by endpoint and request parameters:

```text
location:provinces
location:regencies:{province_code}:{code_format}
location:districts:{regency_code}:{code_format}
location:villages:{district_code}:{code_format}
location:search:{query_hash}:{limit}
```

Default TTL:

```env
LOCATION_CACHE_TTL=4320h
```

`4320h` equals 180 days, or approximately six months.

## Docker

Build and run database, Redis, backend API, and frontend:

```bash
docker compose up --build
```

Local URLs:

```text
Backend API: http://localhost:8080
Frontend:    http://localhost:8081
```

Images are separated:

| Image | Dockerfile | Purpose |
|-------|------------|---------|
| Backend | `Dockerfile` | Go API, migrations, seed file |
| Frontend | `frontend/Dockerfile` | Nginx static frontend |

The frontend Nginx proxy forwards `/api/*` and `/healthz` to the backend using `API_BASE_URL`.

## Frontend

Live demo:

```text
https://location-service-do.vercel.app/
```

The frontend is intentionally static. It can be hosted on Vercel, Nginx, object storage, or opened through a local static server. It supports:

- service health check
- province, regency, district, and village browsing
- short code toggle
- search
- API response inspector
- configurable API base URL

## Postman

Import:

```text
postman/location-service.postman_collection.json
```

## Verification

```bash
go test ./...
go build ./...
```


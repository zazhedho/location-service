# Location Service

Lightweight read-only API for Indonesian administrative locations. Data comes from `wilayah.sql`, then gets normalized into provinces, regencies, districts, and villages.

## Why This Exists

Projects should not depend on external location APIs at runtime. This service becomes the internal source of truth and keeps external/import concerns in one place.

## Stack

- Go standard library HTTP server
- Go 1.26
- PostgreSQL
- `github.com/lib/pq`

Starter-kit style package layout, without starter-kit auth/RBAC dependencies. Keep this service small and boring.

## Project Structure

```text
.
├── cmd/importer/                 # Bulk import command logic
├── infrastructure/database/      # PostgreSQL connection and migration runner
├── internal/domain/location/     # Location entity and import stats
├── internal/handlers/http/       # HTTP handlers
├── internal/interfaces/location/ # Service and repository contracts
├── internal/repositories/        # SQL queries
├── internal/router/              # Route registration
├── internal/services/            # Validation and use-case logic
├── migrations/                   # SQL schema
├── pkg/messages/                 # Shared response messages
├── pkg/response/                 # Starter-kit style response envelope
├── postman/                      # Postman collection
├── utils/                        # Env helpers
└── main.go                       # Command entrypoint
```

## Data Model

Source file format:

```text
11                 province
11.01              regency/city
11.01.01           district
11.01.01.2001      village
```

Normalized tables:

- `raw_locations`
- `provinces`
- `regencies`
- `districts`
- `villages`

Imported data count from the current `wilayah.sql`:

| Table | Rows |
|-------|------|
| `raw_locations` | 91,599 |
| `provinces` | 38 |
| `regencies` | 514 |
| `districts` | 7,285 |
| `villages` | 83,762 |

## Configuration

Create `.env` from the example:

```bash
cp .env.example .env
```

Main variables:

```env
APP_ENV=development
PORT=8080
PATH_MIGRATE=migrations/000001_init.sql
COMMAND=serve
DATABASE_URL=postgres://location:location@localhost:5438/location?sslmode=disable
REDIS_HOST=localhost
REDIS_PORT=6379
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

Docker entrypoint options:

```env
RUN_MIGRATION=false
RUN_IMPORT=false
IMPORT_FILE=/app/data/wilayah.sql
```

Runtime read flow:

```text
Redis -> PostgreSQL -> response
```

If Redis is unavailable, the service logs the error and continues with PostgreSQL only. `LOCATION_CACHE_TTL=4320h` keeps cached location responses for six months.

## Quick Start

Start PostgreSQL and Redis:

```bash
docker compose up -d postgres redis
```

Run migration:

```bash
go run . migrate
```

Import data from workspace root:

```bash
go run . import -file ../wilayah.sql
```

Start API:

```bash
go run . serve
```

API runs at:

```text
http://localhost:8080
```

## Commands

```bash
go run . migrate
```

Creates or updates database schema.

```bash
go run . import -file ../wilayah.sql
```

Imports `wilayah.sql` into `raw_locations`, then bulk loads normalized tables. Import truncates existing location tables by default.

```bash
go run . serve
```

Starts read-only HTTP API.

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

Location item:

```json
{
  "code": "11.01",
  "full_code": "11.01",
  "name": "Kabupaten Aceh Selatan",
  "level": "regency",
  "parent_code": "11"
}
```

Field notes:

- `code`: follows `code_format`; full code by default, short code when `code_format=short`.
- `full_code`: always contains complete administrative code.
- `level`: one of `province`, `regency`, `district`, `village`.
- `parent_code`: parent full code. Empty for provinces and search results.

## Endpoints

### Healthcheck

```text
GET /healthz
```

Response:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 200,
  "status": true,
  "message": "Success",
  "data": {
    "status": "ok"
  }
}
```

### Provinces

```text
GET /api/locations/provinces
```

Example response:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 200,
  "status": true,
  "message": "Success",
  "data": [
    {
      "code": "11",
      "full_code": "11",
      "name": "Aceh",
      "level": "province"
    }
  ]
}
```

### Regencies

```text
GET /api/locations/regencies?province_code=11
GET /api/locations/regencies?province_code=11&code_format=short
```

Full code response item:

```json
{
  "code": "11.01",
  "full_code": "11.01",
  "name": "Kabupaten Aceh Selatan",
  "level": "regency",
  "parent_code": "11"
}
```

Short code response item:

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

Example short-code response item:

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

Example short-code response item:

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

Example response:

```json
{
  "log_id": "019ab0f0-c8ec-7d25-9f62-2f23d92fcda3",
  "code": 200,
  "status": true,
  "message": "Success",
  "data": [
    {
      "code": "11",
      "full_code": "11",
      "name": "Aceh",
      "level": "province"
    }
  ]
}
```

## Code Format

Default API responses use full administrative codes:

```text
province: 11
regency: 11.01
district: 11.01.01
village: 11.01.01.2001
```

For existing clients that still expect short child codes, pass `code_format=short`:

```text
regency: 01
district: 01
village: 2001
```

`full_code` is always included, so clients can migrate from short code to full code gradually.

## Docker

Build and run database, backend API, and frontend:

```bash
docker compose up --build
```

Local URLs:

```text
Backend API: http://localhost:8080
Frontend:    http://localhost:8081
```

The backend image runs:

```text
/app/location-service serve
```

The frontend image is built from `frontend/Dockerfile` and serves static files through Nginx. Nginx proxies `/api/*` and `/healthz` to the backend using `API_BASE_URL`.

Optional startup actions:

```env
RUN_MIGRATION=true
RUN_IMPORT=true
IMPORT_FILE=/app/data/wilayah.sql
AUTO_SEED=true
AUTO_SEED_REQUIRED=false
SEED_FILE=data/wilayah.sql
```

On `serve`, `AUTO_SEED=true` checks `raw_locations`. If the table is empty, the service imports `SEED_FILE` in one transaction using PostgreSQL `COPY`; if data already exists, startup skips seeding. The Docker image includes `data/wilayah.sql`, so first deploy can seed without mounting an extra file.

`RUN_IMPORT=true` remains available for forced manual import from `IMPORT_FILE`.

## Postman

Import this collection:

```text
postman/location-service.postman_collection.json
```

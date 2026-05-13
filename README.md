# Location Service

<p align="center">
  <strong>API wilayah Indonesia untuk provinsi, kabupaten/kota, kecamatan, dan desa/kelurahan.</strong>
</p>

<p align="center">
  <a href="https://location-service-do.vercel.app/"><img alt="Live Demo" src="https://img.shields.io/badge/live-demo-2563eb?style=for-the-badge"></a>
  <a href="https://location-service-y7si.onrender.com/healthz"><img alt="API Health" src="https://img.shields.io/badge/api-health-16a34a?style=for-the-badge"></a>
  <img alt="Go" src="https://img.shields.io/badge/go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/postgresql-ready-4169E1?style=for-the-badge&logo=postgresql&logoColor=white">
  <img alt="Redis" src="https://img.shields.io/badge/redis-cache-DC382D?style=for-the-badge&logo=redis&logoColor=white">
</p>

Location Service menyediakan API wilayah Indonesia siap pakai untuk mengambil data provinsi, kabupaten/kota, kecamatan, dan desa/kelurahan. API ini cocok untuk form alamat, dropdown lokasi berjenjang, validasi kode wilayah, pencarian nama daerah, dan kebutuhan master data wilayah Indonesia.

Dengan HTTP API sederhana, aplikasi bisa mengambil data wilayah Indonesia dari sumber data yang stabil, mencari lokasi berdasarkan nama, serta memilih format kode lengkap atau kode pendek sesuai kebutuhan integrasi.

[Try the Frontend](https://location-service-do.vercel.app/) · [Health Check](https://location-service-y7si.onrender.com/healthz) · [Postman Collection](postman/location-service.postman_collection.json)

## Overview

`location-service` is a read-only Indonesian administrative area API. It loads source data into PostgreSQL tables, caches responses in Redis, and exposes endpoints for location lookup and search.

Common use cases:

- province and city dropdowns
- cascading address forms
- district and village lookup
- location search by name
- internal master data for Indonesian administrative regions

Runtime read flow:

```text
Client -> HTTP API -> Redis -> PostgreSQL -> Response
```

If Redis is unavailable, the API logs the error and continues with PostgreSQL only.

## Interface Preview

The frontend is a lightweight console for trying the API directly from the browser.

| Browse Locations | Search Locations |
|------------------|------------------|
| Pick a province, drill down to regencies, districts, and villages, then inspect the last API response. | Search by location name or code, set result limits, and compare returned levels in one table. |

```text
┌──────────────────────┬──────────────────────────────────────────────┐
│ Location Service     │ Browse Locations                             │
│ Indonesia data       │ Select a province and drill down to villages │
│                      │                                              │
│ ● Service healthy    │ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│                      │ │ Provinces│ │ Regencies│ │ Districts│ ...  │
│ Browse               │ └──────────┘ └──────────┘ └──────────┘      │
│ Search               │                                              │
│                      │ ┌────────────────┐ ┌──────────────────────┐ │
│                      │ │ Provinces      │ │ Child Locations      │ │
│                      │ │ 11 Aceh        │ │ 11.01 Simeulue      │ │
│                      │ │ 12 Sumatera... │ │ 11.02 Aceh Singkil  │ │
│                      │ └────────────────┘ └──────────────────────┘ │
└──────────────────────┴──────────────────────────────────────────────┘
```

### UI Highlights

| Area | Purpose |
|------|---------|
| Sidebar navigation | Switch between browse mode and search mode. |
| Health indicator | Shows backend availability before users test endpoints. |
| Cascading browser | Moves from province to regency, district, and village data. |
| Short code toggle | Switches between full administrative codes and short codes. |
| Response drawer | Displays the latest JSON response with copy support. |
| Responsive layout | Sidebar collapses on small screens for mobile testing. |

## Features

- Province, regency/city, district, and village lookup APIs.
- Location count stats for dashboard cards and first-load summaries.
- Search API across all location levels.
- PostgreSQL normalized tables for stable source-of-truth data.
- Redis cache before database reads, with default TTL of six months.
- One-time automatic seed on first startup when `raw_locations` is empty.
- Consistent JSON response envelope for success and error responses.
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
├── data/                         # Bundled seed data
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
├── pkg/response/                 # Shared response envelope
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

Seed and import variables are available in `.env.example`.

`AUTO_SEED=true` runs during `serve`. It checks `raw_locations`; if the table is empty, it imports the configured seed file in one transaction using PostgreSQL `COPY`. Startup skips seeding when data already exists.

## Quick Start

Start PostgreSQL and Redis:

```bash
docker compose up -d postgres redis
```

Run migration:

```bash
go run . migrate
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
go run . import -file <path-to-location-data.sql>
```

Imports a location data SQL file into `raw_locations`, then bulk-loads normalized tables. Import truncates existing location tables by default.

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

### Stats

```text
GET /api/locations/stats
GET /api/locations/stats?province_code=11
GET /api/locations/stats?regency_code=11.01
GET /api/locations/stats?district_code=11.01.01
```

Example data:

```json
{
  "raw": 91599,
  "provinces": 38,
  "regencies": 514,
  "districts": 7285,
  "villages": 83762,
  "total": 91600
}
```

The frontend uses the global stats for the main overview cards. When a province, regency, or district is selected, it requests scoped stats so the current selection panel shows local counts without changing the global overview.

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

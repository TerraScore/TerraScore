# LandIntel

Land intelligence and field verification platform. Connects landowners with field agents (Rapido-style gig model) for on-ground surveys, risk assessments, and property reports.

```
┌─────────────┐     ┌───────┐     ┌──────────────────┐     ┌────────────┐
│  Next.js Web │────▶│ Kong  │────▶│  Go Monolith     │────▶│ PostgreSQL │
│  Dashboard   │     │  GW   │     │  (Chi + sqlc)    │     │ + PostGIS  │
└─────────────┘     └───────┘     └──────┬───────────┘     └────────────┘
┌─────────────┐         │               │                   ┌────────────┐
│ React Native│─────────┘               ├──────────────────▶│   Redis    │
│ Agent App   │                         │                   └────────────┘
└─────────────┘                         │                   ┌────────────┐
                                        ├──────────────────▶│  S3/MinIO  │
                                        │                   └────────────┘
                                        │                   ┌────────────┐
                                        └──────────────────▶│  Keycloak  │
                                                            └────────────┘
```

## Tech Stack

| Layer     | Technology                                            |
|-----------|-------------------------------------------------------|
| Backend   | Go 1.24, Chi router, sqlc, pgx, gorilla/websocket    |
| Web       | Next.js 14, React 18, Tailwind CSS, Mapbox GL, Zustand |
| Mobile    | React Native / Expo 54, expo-router, Zustand          |
| Database  | PostgreSQL 16 + PostGIS 3.4                           |
| Cache     | Redis 7                                               |
| Identity  | Keycloak 24 (OTP via MSG91)                           |
| Gateway   | Kong 3.6 (DB-less, declarative)                       |
| Storage   | S3 / MinIO (local dev)                                |
| Infra     | AWS ECS Fargate, RDS, ElastiCache, ALB                |

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Go 1.24+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) & npm
- [Expo CLI](https://docs.expo.dev/get-started/installation/) — `npm install -g expo-cli` (mobile only)

## Quick Start

```bash
# 1. Start infrastructure (Postgres, Redis, Keycloak, Kong, MinIO)
docker compose up -d

# 2. Copy env and run migrations
cp .env.example .env
go run ./cmd/migrate -direction up -db "postgres://landintel:landintel@localhost:5432/landintel?sslmode=disable"

# 3. Start the API server
go run ./cmd/server
```

The API will be available at `http://localhost:8080`. Kong proxies on `:8000`, Keycloak on `:8180`, MinIO console on `:9001`.

## Running the Backend API

```bash
# Using Make (starts infra + server)
make dev

# Or manually
go run ./cmd/server
```

### Environment Variables

Copy `.env.example` to `.env`. Key variables:

| Variable             | Default                 | Description                  |
|----------------------|-------------------------|------------------------------|
| `SERVER_PORT`        | `8080`                  | API listen port              |
| `DB_HOST`            | `localhost`             | PostgreSQL host              |
| `DB_PORT`            | `5432`                  | PostgreSQL port              |
| `REDIS_HOST`         | `localhost`             | Redis host                   |
| `KEYCLOAK_BASE_URL`  | `http://localhost:8180` | Keycloak URL                 |
| `OTP_PROVIDER`       | `mock`                  | OTP provider (`mock`/`msg91`)|

## Running the Web Dashboard

```bash
cd web
npm install
npm run dev
```

Opens at `http://localhost:3000`. The dashboard provides landowner parcel management, map-based boundary drawing, and report viewing.

## Running the Mobile App

```bash
cd mobile
npm install
npx expo start
```

Scan the QR code with Expo Go (Android/iOS). The agent app handles job offers, GPS navigation, photo capture, and survey submission.

## Database

### Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create a new migration
make migrate-create
# Prompts for name, creates up/down SQL files in db/migrations/
```

### sqlc Code Generation

After editing SQL queries in `db/queries/`:

```bash
make sqlc
# Regenerates Go code in db/sqlc/
```

### Connect via psql

```bash
psql "postgres://landintel:landintel@localhost:5432/landintel"
```

## Testing

### Backend

```bash
# Run all tests
make test

# With race detector and coverage
make test-cover
# Opens coverage.html in browser
```

### Web

```bash
cd web
npm run lint    # ESLint
npm run build   # Type-check + build
```

### Linting

```bash
make lint       # go vet + golangci-lint (if installed)
```

## Docker

### Full Stack (dev)

```bash
docker compose up -d      # Start all services
docker compose down        # Stop all services
docker compose logs -f     # Tail logs
```

### Build API Image

```bash
make docker-build
# Or directly:
docker build -t landintel-api:latest .
```

## CI/CD Pipeline

The GitHub Actions pipeline (`.github/workflows/ci.yml`) runs on every push to `main` and on pull requests:

| Job            | Trigger          | Description                                      |
|----------------|------------------|--------------------------------------------------|
| `test-backend` | push + PRs       | Go vet, build, migrations, tests (with Postgres + Redis) |
| `build-web`    | push + PRs       | npm ci, lint, Next.js build                      |
| `docker`       | push to main     | Build and push image to GHCR                     |
| `deploy`       | push to main     | Placeholder for ECS Fargate deployment            |

## API Endpoints

All routes are prefixed with `/v1` unless noted. Protected routes require a `Bearer` JWT token.

| Method | Path                              | Auth     | Description                  |
|--------|-----------------------------------|----------|------------------------------|
| GET    | `/health`                         | None     | Health check                 |
| GET    | `/ws`                             | Query    | WebSocket connection         |
| POST   | `/v1/auth/register`               | None     | Register user                |
| POST   | `/v1/auth/login`                  | None     | Request OTP                  |
| POST   | `/v1/auth/verify-otp`             | None     | Verify OTP, get tokens       |
| POST   | `/v1/auth/refresh`                | None     | Refresh access token         |
| POST   | `/v1/parcels`                     | JWT      | Create parcel                |
| GET    | `/v1/parcels`                     | JWT      | List parcels                 |
| GET    | `/v1/parcels/{id}`                | JWT      | Get parcel details           |
| PUT    | `/v1/parcels/{id}/boundary`       | JWT      | Update parcel boundary       |
| DELETE | `/v1/parcels/{id}`                | JWT      | Delete parcel                |
| POST   | `/v1/agents/register`             | None     | Register agent               |
| GET    | `/v1/agents/me`                   | JWT      | Get agent profile            |
| PUT    | `/v1/agents/me/profile`           | JWT      | Update agent profile         |
| POST   | `/v1/agents/me/location`          | JWT      | Update agent location        |
| PUT    | `/v1/agents/me/availability`      | JWT      | Toggle availability          |
| PUT    | `/v1/agents/me/fcm-token`         | JWT      | Update FCM token             |
| GET    | `/v1/agents/me/jobs`              | Agent    | List agent's jobs            |
| GET    | `/v1/agents/me/offers`            | Agent    | List pending offers          |
| POST   | `/v1/jobs/{id}/accept`            | Agent    | Accept job offer             |
| POST   | `/v1/jobs/{id}/decline`           | Agent    | Decline job offer            |
| GET    | `/v1/jobs/{id}`                   | Agent    | Get job details              |
| POST   | `/v1/jobs/{id}/arrive`            | Agent    | Mark arrival at site         |
| GET    | `/v1/jobs/{id}/media/presigned`   | Agent    | Get presigned upload URL     |
| POST   | `/v1/jobs/{id}/media`             | Agent    | Record uploaded media         |
| POST   | `/v1/jobs/{id}/survey`            | Agent    | Submit survey answers        |
| GET    | `/v1/jobs/{id}/template`          | Agent    | Get survey template          |
| GET    | `/v1/alerts`                      | JWT      | List alerts                  |
| GET    | `/v1/alerts/unread/count`         | JWT      | Get unread count             |
| PUT    | `/v1/alerts/{id}/read`            | JWT      | Mark alert as read           |
| PUT    | `/v1/alerts/read-all`             | JWT      | Mark all alerts as read      |
| GET    | `/v1/parcels/{parcelId}/reports`  | JWT      | List reports for parcel      |
| GET    | `/v1/reports/{id}/download`       | JWT      | Download report              |

## Project Structure

```
.
├── cmd/
│   ├── server/          # API entrypoint
│   └── migrate/         # Migration runner
├── db/
│   ├── migrations/      # SQL migration files (001-009)
│   ├── queries/         # sqlc query definitions
│   └── sqlc/            # Generated Go code
├── internal/
│   ├── auth/            # Authentication, Keycloak, OTP, JWT middleware
│   ├── land/            # Parcel CRUD, boundary validation
│   ├── agent/           # Agent registration, profile, location
│   ├── job/             # Job lifecycle, matching, dispatcher, surveys
│   ├── notification/    # Alerts, in-app notifications
│   ├── report/          # PDF report generation
│   ├── survey/          # Survey templates and responses
│   ├── qa/              # Quality assurance checks
│   ├── billing/         # Billing (Phase 2)
│   ├── ws/              # WebSocket hub
│   └── platform/        # Shared: config, DB, Redis, S3, logger, middleware
├── web/                 # Next.js 14 landowner dashboard
├── mobile/              # React Native / Expo agent app
├── infra/
│   ├── keycloak/        # Realm import JSON
│   └── kong/            # Declarative Kong config
├── docker-compose.yml   # Local dev infrastructure
├── Dockerfile           # Multi-stage Go build
├── Makefile             # Dev commands
└── .github/workflows/   # CI/CD pipeline
```

## License

Proprietary. All rights reserved.

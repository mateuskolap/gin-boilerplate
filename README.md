# Gin Boilerplate

API boilerplate in Go using Gin, GORM, PostgreSQL, Redis and a clean/hexagonal structure.

## Local setup

1. Copy `.env.example` to `.env` and set `JWT_SECRET` with at least 32 characters.
2. Start PostgreSQL and Redis with `docker compose up -d`.
3. Export the migration connection URL and apply the versioned migrations:

```sh
export DATABASE_URL='postgres://postgres:postgres@localhost:5438/boilerplate?sslmode=disable'
atlas migrate apply --env local
```

4. Start the API:

```sh
go run ./cmd/api
```

To create the initial roles, permissions and administrator, set a strong `ADMIN_PASSWORD` and run:

```sh
go run ./cmd/api -seed-only
```

Schema changes must be represented by a new file in `migrations/`; application startup never mutates the database schema.

The initial migration targets an empty PostgreSQL 18 database. If a development database was previously created by GORM `AutoMigrate`, back it up and either recreate that development database or baseline it with Atlas only after confirming that its schema matches the migration; applying the initial migration directly over existing tables will fail.

## Operational endpoints

- `GET /health/live` checks whether the process is alive.
- `GET /health/ready` checks PostgreSQL and Redis connectivity.
- Swagger is available at `/swagger/index.html` outside production.

## Authorization

PostgreSQL is the source of truth for roles and permissions. JWT access tokens contain identity and registered claims only; protected operations check the current role-permission relationships directly in the database.

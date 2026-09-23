# Gin Boilerplate

API boilerplate in Go using Gin, GORM, PostgreSQL, Redis and a clean/hexagonal structure.

## Local setup

1. Copy `.env.example` to `.env` and set `JWT_SECRET` with at least 32 characters.
2. Start PostgreSQL and Redis with `docker compose up -d`.
3. Apply the versioned migrations:

```sh
go run ./cmd/migrate up
```

4. Start the API:

```sh
go run ./cmd/api
```

To create the initial roles, permissions and administrator, set a strong `ADMIN_PASSWORD` and run:

```sh
go run ./cmd/api -seed-only
```

## Operational endpoints

- `GET /health/live` checks whether the process is alive.
- `GET /health/ready` checks PostgreSQL and Redis connectivity.
- Swagger is available at `/swagger/index.html` outside production.

## Authorization

PostgreSQL is the source of truth for roles and permissions. JWT access tokens contain identity and registered claims only; protected operations check the current role-permission relationships directly in the database.

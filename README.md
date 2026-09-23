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

## Local storage

The application initializes one private local storage disk. Configure its root and maximum size per file in `.env`:

```dotenv
STORAGE_ROOT=./storage/private
STORAGE_MAX_FILE_SIZE_BYTES=10485760
```

Use `port.Storage` as a dependency of the use case that owns a file. The use case chooses a relative key such as `users/<user-id>/avatar.webp` and may enforce a smaller size limit before calling `Put`. The storage service provides `Put`, `Open`, `Delete`, `Exists`, and `Stat`. `Open` returns a reader that the caller must close. `Put` rejects existing keys and files above the configured limit; `Delete` succeeds when the key is absent. Use `errors.Is` with the sentinel errors in `internal/domain/port/storage.go` to handle expected failures.

Store the relative key in the owning entity's database table when that entity has one file, for example `users.avatar_path`. A domain-specific table is appropriate when files have their own metadata or multiple relationships. Do not store an absolute filesystem path or public URL as the file reference.

In the Docker image, the default root is `/app/storage/private` and is owned by UID `10001`. Mount a persistent volume at `/app/storage` when running the API in a container. A host-mounted directory must be writable by UID `10001`.

An interrupted process may leave files with the `.storage-tmp-` prefix in storage directories. Stop the application before removing these temporary files manually.

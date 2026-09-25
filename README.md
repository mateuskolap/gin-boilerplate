# Gin Boilerplate

API boilerplate in Go using Gin, GORM, PostgreSQL, Redis and a clean/hexagonal structure.

## Local setup

1. Copy `.env.example` to `.env` and set `JWT_SECRET` with at least 32 characters.
2. Start PostgreSQL and Redis with `docker compose up -d`.
3. Apply the versioned migrations using the project command. It loads the existing `DB_*` settings from `.env`:

```sh
make migrate-up
```

4. Start the API:

```sh
go run ./cmd/api
```

## Queues and scheduled tasks

Background work uses [Asynq](https://github.com/hibiken/asynq) backed by Redis.
The API, queue worker, and scheduler are separate processes:

```sh
go run ./cmd/api
make queue-worker
make queue-scheduler
```

Run exactly one scheduler replica for each environment. It registers the
recurring tasks declared in code and enqueues them in UTC; missed executions
are not replayed after scheduler downtime. Workers can be scaled independently.

Queue data uses `QUEUE_REDIS_DB=1` by default, separate from cache and rate
limit data in `REDIS_DB=0`. The worker handles the `default` and `maintenance`
queues with weighted priority. Queue failures, retries, and archived tasks are
written as JSON to `storage/logs/queue.log`; this log does not rotate.

Use cases receive `port.QueueDispatcher` and choose their own retry policy at
dispatch time. For example:

```go
_, err := queue.Dispatch(ctx, port.QueueTask{
    Type: "email.send",
    Payload: json.RawMessage(`{"user_id":"..."}`),
}, port.DispatchOptions{
    Queue: "default",
    Timeout: 30 * time.Second,
    Retry: port.RetryPolicy{
        MaxRetries: 5,
        Backoff: port.RetryBackoffExponential,
        InitialDelay: time.Minute,
        MaxDelay: time.Hour,
    },
})
```

Register the corresponding `port.TaskHandler` in the worker bootstrap. Unknown
task types and invalid JSON payloads are archived without retry.

The included maintenance task removes refresh tokens that have been expired for
`REFRESH_TOKEN_RETENTION` (30 days by default). It runs daily at 03:00 UTC.

Use the official CLI for operational inspection and recovery:

```sh
go install github.com/hibiken/asynq/tools/asynq@v0.26.0
asynq -u 127.0.0.1:6379 -n 1 queue ls
asynq -u 127.0.0.1:6379 -n 1 task ls --queue=maintenance --state=archived
```

The API image starts `/api`. Deploy the worker and scheduler from the same
image by overriding the entrypoint with `/worker` and `/scheduler`.

To create the initial roles, permissions and administrator, set a strong `ADMIN_PASSWORD` and run:

```sh
go run ./cmd/api -seed-only
```

## Database migrations

Migrations are PostgreSQL SQL files in `db/migrations`, managed by `golang-migrate`. The project command reads database settings from `.env` through the same configuration loader as the API. Each migration is a pair of files with the same UTC timestamp prefix:

```text
YYYYMMDDHHMMSS_name.up.sql
YYYYMMDDHHMMSS_name.down.sql
```

Create a new pair with the project generator:

```sh
make migrate-create NAME=add-user-avatar
```

The generator normalizes the name to `snake_case`, creates both files with a SQL comment, and rejects a timestamp collision. It accepts letters, numbers, spaces, hyphens, and underscores.

Apply all migrations, roll back one (or a specified number), or inspect the current version:

```sh
make migrate-up
make migrate-down
make migrate-down STEPS=2
make migrate-version
```

If a migration fails, inspect the database state and migration version before taking recovery action.

## Operational endpoints

- `GET /health/live` checks whether the process is alive.
- `GET /health/ready` checks PostgreSQL and Redis connectivity.
- Swagger is available at `/swagger/index.html` outside production.

Regenerate the Swagger files after changing the API annotations with:

```sh
make swagger
```

## Authorization

PostgreSQL is the source of truth for roles and permissions. JWT access tokens contain identity and registered claims only; protected operations check the current role-permission relationships directly in the database.

## Local storage

The application initializes one private local storage disk. Configure its root in `.env`:

```dotenv
STORAGE_ROOT=./storage/private
```

Use `port.Storage` as a dependency of the use case that owns a file. The use case chooses a relative key such as `users/<user-id>/avatars/<image-id>.jpg` and passes its size limit to `Put` through `port.PutOptions`. The storage service provides `Put`, `Open`, and `Delete`. `Open` returns a reader that the caller must close. `Put` rejects existing keys and files above the supplied limit; `Delete` succeeds when the key is absent. Use `errors.Is` with the sentinel errors in `internal/domain/port/storage.go` to handle expected failures.

Store the relative key in the owning entity's database table when that entity has one file, for example `users.avatar_key`. A domain-specific table is appropriate when files have their own metadata or multiple relationships. Do not store an absolute filesystem path or public URL as the file reference.

In the Docker image, the default root is `/app/storage/private` and is owned by UID `10001`. Mount a persistent volume at `/app/storage` when running the API in a container. A host-mounted directory must be writable by UID `10001`.

An interrupted process may leave files with the `.storage-tmp-` prefix in storage directories. Stop the application before removing these temporary files manually.

## Error logs

The API creates `storage/logs/app.log` when it starts. Errors are appended as one JSON object per line, including HTTP 5xx failures and recovered panics. Normal requests and expected HTTP 4xx errors are not written to the file. Console logs continue to include operational messages and requests.

The log file does not rotate automatically. Keep `storage/logs` on a persistent volume if logs must survive container replacement, and arrange rotation externally when needed. The application must be able to write to this directory; if it cannot open the file, startup fails.

.PHONY: swagger migrate-create migrate-up migrate-down migrate-version queue-worker queue-scheduler

swagger:
	go tool swag init -d ./cmd/api,./internal/delivery/http -g main.go -o docs --parseInternal

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "usage: make migrate-create NAME=<migration-name>" >&2; exit 2; fi
	go run ./cmd/migration create "$(NAME)"

migrate-up:
	go run ./cmd/migration up

migrate-down:
	go run ./cmd/migration down $(STEPS)

migrate-version:
	go run ./cmd/migration version

queue-worker:
	go run ./cmd/worker

queue-scheduler:
	go run ./cmd/scheduler

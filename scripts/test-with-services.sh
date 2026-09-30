#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
project="gin-boilerplate-test-$$"

compose() {
	docker compose --project-name "$project" --file docker-compose.test.yaml "$@"
}

cleanup() {
	status=$?
	trap - EXIT
	if ! compose down --volumes --remove-orphans; then
		status=1
	fi
	exit "$status"
}
trap cleanup EXIT

compose up --detach --wait --wait-timeout 90
postgres_port=$(compose port postgres 5432 | sed 's/.*://;q')
redis_port=$(compose port redis 6379 | sed 's/.*://;q')

TEST_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:${postgres_port}/test?sslmode=disable" \
	TEST_REDIS_ADDR="127.0.0.1:${redis_port}" \
	TEST_REDIS_DB=14 \
	TEST_QUEUE_REDIS_DB=15 \
	"$@"

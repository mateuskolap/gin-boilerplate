#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p tmp
go build -o tmp/app ./cmd/app
exec ./tmp/app "$@"

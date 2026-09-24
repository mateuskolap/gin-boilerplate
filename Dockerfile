FROM golang:1.27-alpine AS build

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 10001 -g '' appuser

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server ./cmd/api

RUN mkdir -p /app/storage/private

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=build /etc/passwd /etc/passwd

COPY --from=build /etc/group /etc/group

WORKDIR /app

COPY --from=build --chown=10001:10001 /app/storage /app/storage

USER 10001:10001

COPY --from=build /app/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]

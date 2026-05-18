FROM golang:1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 go build \
    -ldflags "-X moontracer/internal/commands.BuildTime=${BUILD_TIME}" \
    -o /moontracer ./cmd/moontracer

FROM alpine:3.21
RUN apk add --no-cache ca-certificates

RUN mkdir -p /app/data
WORKDIR /app

COPY --from=build /moontracer /app/moontracer

ENTRYPOINT ["/app/moontracer"]
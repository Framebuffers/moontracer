FROM golang:1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /moontracer ./cmd/moontracer

FROM alpine:3.21
RUN apk add --no-cache ca-certificates

RUN adduser -D -h /app moontracer
RUN mkdir -p /app/data && chown moontracer:moontracer /app/data
WORKDIR /app

COPY --from=build --chown=moontracer:moontracer /moontracer /app/moontracer

RUN printf '#!/bin/sh\nset -e\n\
chown -R moontracer:moontracer /app/data\n\
exec su -s /bin/sh moontracer -c "/app/moontracer"\n' > /docker-entrypoint.sh \
    && chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
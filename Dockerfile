FROM golang:1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /moontracer ./cmd/moontracer

FROM alpine:3.21
RUN apk add --no-cache ca-certificates

RUN mkdir -p /app/data && chown moontracer:moontracer /app/data
RUN adduser -D -h /app moontracer
WORKDIR /app

COPY --from=build --chown=moontracer:moontracer /moontracer /app/moontracer

COPY <<'EOF' /docker-entrypoint.sh
#!/bin/sh
set -e

# Read Docker secrets into environment variables.
if [ -f /run/secrets/discord_bot_token ]; then
  export DISCORD_TOKEN="$(cat /run/secrets/discord_bot_token)"
fi

if [ -f /run/secrets/discord_guild_id ]; then
  export DISCORD_GUILD_ID="$(cat /run/secrets/discord_guild_id)"
fi

exec su -s /bin/sh moontracer -c "/app/moontracer"
EOF
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

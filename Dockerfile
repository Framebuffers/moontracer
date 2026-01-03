FROM oven/bun:1-alpine AS base

FROM base AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN bun run build

FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# Install runtime dependencies
RUN apk add --no-cache dumb-init su-exec

COPY --from=builder --chown=bun:bun /app/public ./public
COPY --from=builder --chown=bun:bun /app/.next/standalone ./
COPY --from=builder --chown=bun:bun /app/.next/static ./.next/static

# Create entrypoint
COPY <<'EOF' /docker-entrypoint.sh
#!/bin/sh
set -e

if [ -d /run/secrets ]; then
  chmod 644 /run/secrets/* 2>/dev/null || true
fi

exec su-exec bun bun server.js "$@"
EOF

RUN chmod +x /docker-entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/docker-entrypoint.sh"]
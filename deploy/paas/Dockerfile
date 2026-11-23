# syntax=docker/dockerfile:1.7
# One image for every managed container platform (Fly.io, Cloudflare Containers,
# Render, Railway, Koyeb, ...). Build from the repository root:
#
#   docker build -f deploy/paas/Dockerfile -t hydrogo-paas .
#
# SQLite by default, replicated to S3/R2 with Litestream when LITESTREAM_REPLICA_URL
# is set. Set HYDRO_DATABASE_DIALECT=postgres and HYDRO_DATABASE_DSN to use
# PostgreSQL instead (recommended for commerce). See deploy/paas/entrypoint.sh.

ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM --platform=$BUILDPLATFORM node:22-alpine AS admin-build
WORKDIR /admin
COPY admin/package.json ./
RUN npm install
COPY admin/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=admin-build /admin/dist ./internal/adminui/dist
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/hydro ./cmd/hydro

FROM litestream/litestream:0.3.13 AS litestream

FROM alpine:3.22
# su-exec: the entrypoint starts as root only to fix ownership of a freshly
# mounted volume, then drops to the unprivileged hydro user.
RUN apk add --no-cache ca-certificates tzdata su-exec \
    && addgroup -S hydro \
    && adduser -S -G hydro -h /app hydro \
    && mkdir -p /app/data \
    && chown -R hydro:hydro /app

WORKDIR /app
COPY --from=build /out/hydro /app/hydro
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream
COPY --from=build /src/content-types /app/content-types
COPY deploy/paas/config.yaml /app/config.template.yaml
COPY deploy/paas/entrypoint.sh /app/entrypoint.sh
RUN chmod 0755 /app/entrypoint.sh

ENV PATH="/app:${PATH}"
EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
  CMD wget -q -O - "http://127.0.0.1:${PORT:-8080}/healthz" >/dev/null || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["hydro", "serve", "--config", "/tmp/hydro-config.yaml"]

# Multi-stage reproducible Docker build for HydroGo
FROM golang:1.24-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -o /bin/hydro \
    ./cmd/hydro

# Production runner image (minimal scratch/alpine with non-root security)
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 10001 -S hydrogo && \
    adduser -u 10001 -S hydrogo -G hydrogo -s /sbin/nologin && \
    mkdir -p /var/lib/hydrogo /srv/static && \
    chown -R hydrogo:hydrogo /var/lib/hydrogo /srv/static

COPY --from=builder /bin/hydro /usr/local/bin/hydro

USER 10001:10001
WORKDIR /var/lib/hydrogo

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/hydro"]
CMD ["serve"]

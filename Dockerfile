# syntax=docker/dockerfile:1.7

ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM golang:1.27-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/hydro ./cmd/hydro

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S hydro \
    && adduser -S -G hydro -h /app hydro \
    && mkdir -p /app/data /app/themes /app/migrations \
    && chown -R hydro:hydro /app

WORKDIR /app
COPY --from=build /out/hydro /app/hydro
COPY --from=build /src/hydrogo.lock /app/hydrogo.lock
COPY --from=build /src/themes /app/themes
COPY --from=build /src/config.example.yaml /app/config.example.yaml

USER hydro
VOLUME ["/app/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz >/dev/null || exit 1

# Mount a deployment-owned config at /app/config.yaml. A license file, when
# required by the lock, must also be mounted at the path configured there.
ENTRYPOINT ["/app/hydro"]
CMD ["serve", "--config", "/app/config.yaml"]

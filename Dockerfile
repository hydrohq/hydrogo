# syntax=docker/dockerfile:1.7

ARG TARGETOS=linux
ARG TARGETARCH=amd64

# The two build stages run on the machine doing the build and cross-compile for
# the target (Go with CGO off; the admin bundle is static), so building the arm64
# image does not run Node and the Go toolchain under emulation. Only the final
# stage below runs for the target architecture.
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
    go build -trimpath -ldflags="-s -w" \
    -o /out/hydro ./cmd/hydro

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S hydro \
    && adduser -S -G hydro -h /app hydro \
    && mkdir -p /app/data /app/migrations \
    && chown -R hydro:hydro /app

WORKDIR /app
COPY --from=build /out/hydro /app/hydro
COPY --from=build /src/hydrogo.lock /app/hydrogo.lock
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

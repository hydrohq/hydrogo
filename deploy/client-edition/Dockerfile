# Client Edition golden path: build the selected edition as one static binary.
FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags='-s -w -buildid=' -o /out/hydro ./cmd/hydro

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/hydro /hydro
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/hydro"]
CMD ["serve"]

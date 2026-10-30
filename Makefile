GO ?= go
NPM ?= npm
BUILD_DIR ?= bin
GOOS ?= linux
GOARCH ?= amd64

.PHONY: build build-hosting build-vps test check clean admin-build admin-test admin-check bench

admin-build:
	$(NPM) --prefix admin run build

admin-check:
	$(NPM) --prefix admin run check

admin-test:
	$(NPM) --prefix admin test

build: admin-build
	mkdir -p $(BUILD_DIR)
	$(GO) build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $(BUILD_DIR)/hydro ./cmd/hydro

# Shared/CGI hosting: no C toolchain on the target, cross-compile a static,
# pure-Go binary (both the SQLite and MySQL drivers already are CGO-free).
build-hosting: admin-build
	mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $(BUILD_DIR)/hydro-hosting ./cmd/hydro

# VPS/container: same CGO-free build, runs as a long-lived process
# (hydro serve / --fastcgi) under your own process manager instead of being
# spawned per-request by a shared-hosting CGI handler.
build-vps: admin-build
	mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $(BUILD_DIR)/hydro-vps ./cmd/hydro

test:
	$(GO) test ./...

# Everything a change must pass before it is pushed.
check:
	$(GO) run ./tools/check

bench:
	$(GO) run ./tools/bench

clean:
	rm -rf $(BUILD_DIR)

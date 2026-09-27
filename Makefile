GO ?= go
BUILD_DIR ?= bin

.PHONY: build test clean

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $(BUILD_DIR)/hydro ./cmd/hydro

test:
	$(GO) test ./...

clean:
	rm -rf $(BUILD_DIR)

BINARY_NAME := lora-config-sx1262
PKG := lora-config-SX1262/cmd/lora-config-sx1262

TAG       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%d-%m-%Y %H:%M')

LDFLAGS := -X main.Tag=$(TAG) -X main.Commit=$(COMMIT) -X 'main.BuildDate=$(BUILD_DATE)'

.PHONY: build clean test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(PKG)

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

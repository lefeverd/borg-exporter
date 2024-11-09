APP_NAME := borg-exporter
VERSION := $(shell git describe --tags --always)
BUILD_DIR := build

.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME) -ldflags "-X main.Version=$(VERSION)" ./cmd/cli

.PHONY: test
test:
	go test ./internal/...

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: package
package:
	tar -czvf $(BUILD_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) $(APP_NAME)

.PHONY: release
release:
	./release.sh minor

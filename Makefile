.PHONY: all test build-linux package-mac clean run

APP_NAME := lumina-photo-sync
APP_VERSION := 0.9.0
BIN_DIR := bin
SRC_DIR := ./cmd/lumina-photo-sync
LOCAL_LIB_DIR := $(HOME)/.local/lib
LIB_FLAGS := $(if $(wildcard $(LOCAL_LIB_DIR)),CGO_LDFLAGS="-L$(LOCAL_LIB_DIR)") CGO_ENABLED=1

all: test build-linux

test:
	$(LIB_FLAGS) go test -v ./...

build-linux:
	mkdir -p $(BIN_DIR)
	$(LIB_FLAGS) go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(SRC_DIR)
	@echo "Built: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

package-mac:
	./scripts/package-mac.sh

run:
	$(LIB_FLAGS) go run $(SRC_DIR)

clean:
	rm -rf $(BIN_DIR)

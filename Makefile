.PHONY: build install clean test help

BINARY_NAME=daylit
BUILD_DIR=.
INSTALL_PATH=/usr/local/bin

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/daylit

install: build ## Install the binary to system path
	sudo mv $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

clean: ## Remove built binary
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

test: ## Run tests
	go test -v ./...

run: build ## Build and run with example
	@echo "Building and running daylit..."
	./$(BINARY_NAME) --help

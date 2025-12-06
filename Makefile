.PHONY: test test-binance test-bybit test-okx test-gate help example vet lint

help: ## Show help information
	@echo "Available commands:"
	@echo "  make test              - Run all exchange tests (Binance, Bybit, OKX, Gate)"
	@echo "  make test-binance      - Run Binance tests only"
	@echo "  make test-bybit        - Run Bybit tests only"
	@echo "  make test-okx         - Run OKX tests only"
	@echo "  make test-gate         - Run Gate tests only"
	@echo "  make vet               - Run go vet on all packages"
	@echo "  make lint              - Run golangci-lint on all packages"
	@echo "  make example           - Build all example programs to bin directory"
	@echo "  PROXY_URL=xxx make test - Run tests with proxy (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make test"
	@echo "  make test-binance"
	@echo "  make vet"
	@echo "  make lint"
	@echo "  make example"
	@echo "  PROXY_URL=http://127.0.0.1:7890 make test"

test: ## Run all exchange tests
	@echo "Running all exchange tests..."
	@$(MAKE) test-binance
	@$(MAKE) test-bybit
	@$(MAKE) test-okx
	@$(MAKE) test-gate
	@echo "All tests completed!"

test-binance: ## Run Binance tests
	@echo "Running Binance tests..."
	go test -count=1 -v -timeout=60s ./binance/...

test-bybit: ## Run Bybit tests
	@echo "Running Bybit tests..."
	go test -count=1 -v -timeout=60s ./bybit/...

test-okx: ## Run OKX tests
	@echo "Running OKX tests..."
	go test -count=1 -v -timeout=60s ./okx/...

test-gate: ## Run Gate tests
	@echo "Running Gate tests..."
	go test -count=1 -v -timeout=60s ./gate/...

vet: ## Run go vet on all packages
	go vet ./...

lint: ## Run golangci-lint on all packages
	golangci-lint run --timeout=5m

example: ## Build all example programs to bin directory
	@echo "Building example programs..."
	@mkdir -p bin
	@for dir in examples/*/; do \
		name=$$(basename $$dir); \
		echo "Building $$name..."; \
		go build -o bin/$$name ./$$dir; \
	done
	@echo "Done! Binaries are in bin/ directory"

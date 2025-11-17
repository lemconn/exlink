.PHONY: test help example vet

help: ## Show help information
	@echo "Available commands:"
	@echo "  make test              - Run all tests in exchanges directory"
	@echo "  make vet               - Run go vet on all packages"
	@echo "  make example           - Build all example programs to bin directory"
	@echo "  PROXY_URL=xxx make test - Run tests with proxy (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make test"
	@echo "  make vet"
	@echo "  make example"
	@echo "  PROXY_URL=http://127.0.0.1:7890 make test"

test: ## Run tests
	go test -count=1 -v ./exchanges/...

vet: ## Run go vet on all packages
	go vet ./...

example: ## Build all example programs to bin directory
	@echo "Building example programs..."
	@mkdir -p bin
	@for dir in examples/*/; do \
		name=$$(basename $$dir); \
		echo "Building $$name..."; \
		go build -o bin/$$name ./$$dir; \
	done
	@echo "Done! Binaries are in bin/ directory"

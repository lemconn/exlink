.PHONY: test help

help: ## Show help information
	@echo "Available commands:"
	@echo "  make test              - Run all tests in exchanges directory"
	@echo "  PROXY_URL=xxx make test - Run tests with proxy (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make test"
	@echo "  PROXY_URL=http://127.0.0.1:7890 make test"

test: ## Run tests
	go test -count=1 -v ./exchanges/...

.DEFAULT_GOAL := help

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make <target>\n\nTargets:\n"} \
		/^([a-zA-Z_-]+):.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the omnidex binary
	go build -o omnidex ./cmd/omnidex/main.go

test: ## Run unit tests with race detector
	go test --race ./...

lint: ## Run golangci-lint
	golangci-lint run

mocks: ## Generate mocks using mockery
	mockery

tidy: ## Run go mod tidy
	go mod tidy

fmt: ## Format code with gofmt
	gofmt -w .

fields: ## Fix field alignment
	fieldalignment -fix ./...

tailwind: ## Build Tailwind CSS (requires tailwindcss CLI)
	tailwindcss -i static/css/input.css -o static/css/style.css --minify

dev-css: ## Watch and rebuild Tailwind CSS on changes
	tailwindcss -i static/css/input.css -o static/css/style.css --watch

run: build ## Build and run the server
	./omnidex serve --config runtime/config.yml

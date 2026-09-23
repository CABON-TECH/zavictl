.PHONY: build test lint clean check

BINARY_NAME=zavictl
MAIN_PATH=cmd/zavictl/main.go

build:
	@echo "==> Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)

test:
	@echo "==> Running tests..."
	go test -v -race ./...

lint:
	@echo "==> Running linter..."
	go vet ./...

clean:
	@echo "==> Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f demo-workflow.yaml demo-providers.yaml

check: lint test build
	@echo "==> All checks passed!"

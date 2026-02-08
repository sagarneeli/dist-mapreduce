.PHONY: all build clean run-local

all: build

build:
	go build -o bin/coordinator cmd/coordinator/main.go
	go build -o bin/worker cmd/worker/main.go

clean:
	rm -rf bin/
	rm -rf data/output/
	rm -rf data/intermediate/

run-local: build
	mkdir -p data/input data/output data/intermediate
	# Usage: ./bin/coordinator <files>
	@echo "Start coordinator and workers manually for local testing"

fmt:
	go fmt ./...

lint:
	# Tries to run golangci-lint if installed, otherwise warns
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Running basic 'go vet' instead."; \
		go vet ./...; \
	fi

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

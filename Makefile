.PHONY: build test run docker-build docker-up docker-down clean
build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run:
	go run cmd/server/main.go

docker-build:
	docker build -t adstxt-api:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker-compose down -v

lint:
	golangci-lint run ./...

deps:
	go mod download
	go mod tidy

.DEFAULT_GOAL := build
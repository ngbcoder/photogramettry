.PHONY: dev build test lint frontend-dev frontend-build docker

dev:
	go run ./cmd/photogen-web --port 8080 --verbose

build:
	go build -o bin/photogen-web ./cmd/photogen-web
	cd frontend && npm run build

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

test:
	go test ./...

lint:
	golangci-lint run ./...

docker:
	docker build -t photogen-web .

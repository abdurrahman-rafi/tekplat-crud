APP_NAME=tekplat-crud

.PHONY: run build tidy fmt

run:
	go run ./cmd/web

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/web

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal

.PHONY: build test lint clean

APP_NAME=astria

build:
	go build -o bin/$(APP_NAME) cmd/astria/main.go

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

.PHONY: build test lint clean db-up db-down db-logs db-shell

APP_NAME=astria

build:
	go build -o bin/$(APP_NAME) cmd/astria/main.go

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

db-shell:
	docker exec -it astria_postgres psql -U astria_user -d astria_db
.PHONY: build build-migrate clean run-api run-migrate

build:
	go build -o bin/api ./cmd/api
	go build -o bin/migrate ./cmd/migrate

build-migrate:
	go build -o bin/migrate ./cmd/migrate

clean:
	rm -rf bin/

run-api:
	go run ./cmd/api

run-migrate:
	go run ./cmd/migrate

# Development with hot reload
dev-api:
	air -c .air.toml
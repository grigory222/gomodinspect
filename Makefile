.PHONY: build run test lint mock clean docker-build docker-run docker-stop

build:
	go build -o bin/gomodinspect ./cmd/gomodinspect

run: build
	./bin/gomodinspect $(REPO)

test:
	go test ./... -v

mock:
	mockery

lint:
	golangci-lint run ./...

docker-build:
	docker compose build

docker-run:
	docker compose run --rm gomodinspect $(REPO)

docker-stop:
	docker compose down

clean:
	rm -rf bin/

.PHONY: build test vet fmt run docker

BIN := bin/gravity-server

build:
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

run:
	go run ./cmd/server

docker:
	docker build -t gravity-reduction:latest .

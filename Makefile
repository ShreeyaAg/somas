.PHONY: test run build

test:
	go test -race ./...
	go vet ./...

run:
	go run ./cmd/somas

build:
	go build -o bin/somas ./cmd/somas

.PHONY: build test lint install clean

build:
	go build -o bin/equativ-maestro-pp-cli ./cmd/equativ-maestro-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/equativ-maestro-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/equativ-maestro-pp-mcp ./cmd/equativ-maestro-pp-mcp

install-mcp:
	go install ./cmd/equativ-maestro-pp-mcp

build-all: build build-mcp

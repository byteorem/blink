BINARY := blink
PKG := ./cmd/blink

.PHONY: build test lint clean install dev cover fmt

build:
	go build -o $(BINARY) $(PKG)

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY) coverage.out coverage.html

install:
	go install $(PKG)

dev: build
	./$(BINARY)

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	gofmt -w .
	goimports -w .

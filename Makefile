.PHONY: all build test clean lint setup

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=muhtar

all: setup test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/main.go

test:
	$(GOTEST) -v -race -cover ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.txt
	rm -f coverage.html

lint:
	golangci-lint run

setup:
	$(GOMOD) download
	$(GOMOD) verify
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/main.go
	./$(BINARY_NAME)

docker-build:
	docker build -t muhtar .

docker-run:
	docker run -p 8080:8080 muhtar

coverage:
	$(GOTEST) -coverprofile=coverage.txt -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html 
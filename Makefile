GO ?= go
BIN := boardwalk

.PHONY: all build test vet lint check run clean

all: check build

build:
	$(GO) build -o $(BIN) ./cmd/boardwalk

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

lint:
	$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

check: vet lint test

run:
	$(GO) run ./cmd/boardwalk $(ARGS)

clean:
	rm -f $(BIN)

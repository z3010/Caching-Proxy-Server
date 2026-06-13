BINARY=caching-proxy
CMD=./cmd/caching-proxy

.PHONY: build run install clean test

## Build the binary into ./bin/
build:
	go build -o bin/$(BINARY) $(CMD)

## Run directly without building
run:
	go run $(CMD)/main.go $(ARGS)

## Install the binary to $GOPATH/bin (makes it a global CLI command)
install:
	go install $(CMD)

## Remove built binaries
clean:
	rm -rf bin/

## Run tests
test:
	go test ./...

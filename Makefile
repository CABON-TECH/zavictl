.PHONY: all build test clean

all: test build

build:
	go build ./...

test:
	go test -v ./...

clean:
	go clean

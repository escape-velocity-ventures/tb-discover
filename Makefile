.PHONY: build test lint clean

BINARY := tb-manage
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

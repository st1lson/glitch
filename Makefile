VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/st1lson/glitch/internal/cli.Version=$(VERSION)"

.PHONY: build test lint install clean release

build:
	go build $(LDFLAGS) -o bin/glitch ./cmd/glitch

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/glitch

clean:
	rm -rf bin/

release: clean
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/glitch-linux-amd64 ./cmd/glitch
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/glitch-darwin-amd64 ./cmd/glitch
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/glitch-darwin-arm64 ./cmd/glitch
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/glitch-windows-amd64.exe ./cmd/glitch

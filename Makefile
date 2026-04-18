BINARY_NAME := strava-ap
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/djgrove/strava-attackpoint/cmd.version=$(VERSION)"

.PHONY: build clean test dist

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

dist: clean
	mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

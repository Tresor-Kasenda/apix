BINARY  := apix
MODULE  := github.com/Tresor-Kasend/apix
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install test clean lint release dist dist-brew dist-brew-publish dist-snap dist-all

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/apix

install:
	go install $(LDFLAGS) ./cmd/apix

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

# Cross-compilation targets
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/apix

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/apix

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/apix

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/apix

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/apix

build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-arm64.exe ./cmd/apix

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

dist:
	./scripts/release/build-release.sh

dist-brew: dist
	./scripts/release/generate-brew-formula.sh

dist-brew-publish: dist-brew
	./scripts/release/publish-brew-tap.sh

dist-snap: dist
	./scripts/release/build-snap.sh

dist-all:
	./scripts/release/release-artifacts.sh

release:
	goreleaser release --clean

.PHONY: build clean test install

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
BINARY_NAME = devenv
BINARY_PATH = bin/$(BINARY_NAME)

# Build the binary
build:
	mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/devenv-cli

# Install the binary
install: build
	cp $(BINARY_PATH) /usr/local/bin/$(BINARY_NAME)

# Clean build files
clean:
	$(GOCLEAN)
	rm -rf bin/

# Run tests
test:
	$(GOTEST) -v ./...

# Update dependencies
deps:
	$(GOMOD) tidy

# Run lint checks
lint:
	golangci-lint run ./...

# Build with version info
build-release:
	mkdir -p bin
	$(GOBUILD) -ldflags="-X 'github.com/enigma-brain/devenv-engine/cmd/devenv-cli/version.Version=$(shell git describe --tags --always)'" -o $(BINARY_PATH) ./cmd/devenv-cli
# Run inside nix dev shell automatically
set shell := ["nix", "develop", "--command", "bash", "-c"]

# List available recipes
default:
    @just --list

# Build the binary
build:
    go build -o dns-tui ./cmd/dns-tui

# Run the application
run *args:
    go run ./cmd/dns-tui {{args}}

# Run all tests
test *args:
    go test ./... {{args}}

# Run tests with verbose output
test-v:
    go test -v ./...

# Run the linter
lint:
    golangci-lint run ./...

# Format all Go files
fmt:
    gofmt -w .

# Tidy module dependencies
tidy:
    go mod tidy

# Clean build artifacts
clean:
    rm -f dns-tui

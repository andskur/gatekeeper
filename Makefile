
tidy:
	go mod tidy

update:
	go get -u ./...

test:
	go test ./...

tests: test

test-cover:
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

build:
	go build ./...

# The lint target runs golangci-lint to check for common style and code quality issues
lint:
	golangci-lint run ./...

# The lint-install target installs golangci-lint if not already installed
lint-install:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

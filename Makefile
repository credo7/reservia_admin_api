.PHONY: build test clean run swagger

# Build the application
build:
	go build -o bin/reservia-api cmd/server/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/ build/

# Run the application (requires environment variables)
run:
	go run cmd/server/main.go

# Generate Swagger documentation
swagger:
	swag init -g cmd/server/main.go -o docs/

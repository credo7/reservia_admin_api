# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Environment Setup
- `go mod tidy` - Download and organize dependencies
- Copy `env.example` to `.env` and configure environment variables

### Running the Application
- `make run` - Start the Go server (requires environment variables)
- `go run cmd/server/main.go` - Alternative way to start the server
- `make build` - Build the application binary to `bin/reservia-api`
- Requires MongoDB to be running (see env.example for connection string)
- Swagger UI available at `/swagger/index.html`

### Testing
- `make test` - Run all tests with `go test ./...`

### Documentation
- `make swagger` - Generate Swagger documentation using swag
- `swag init -g cmd/server/main.go -o docs/` - Manual Swagger generation

### Build Tasks
- `make clean` - Remove build artifacts from `bin/` and `build/` directories

## Architecture Overview

### Clean Architecture Implementation
This project follows **Clean Architecture** principles with strict separation of concerns:

**Application Entry Point** (`cmd/server/`)
- `main.go` - Application bootstrap with graceful shutdown, dependency injection, and server lifecycle management

**Domain Layer** (`internal/domain/`)
- Pure business entities with no external dependencies:
  - `employee/entity.go` - Employee domain with business rules and permissions
  - `restaurant/entity.go` - Restaurant domain entities
  - `reservation/entity.go` - Reservation business logic
  - `auth/entity.go` - Authentication domain
  - `city/entity.go` - City data structures

**Repository Layer** (`internal/repository/`)
- `interfaces.go` - Repository contracts defining data access patterns
- `mongodb/` - MongoDB implementations of repository interfaces
- Key principle: Use cases depend on interfaces, not concrete implementations

**Use Case Layer** (`internal/usecase/`)
- Application business rules organized by domain:
  - `employee/` - Employee management use cases
  - `restaurant/` - Restaurant operations
  - `city/` - City data operations
  - `auth/` - Authentication workflows (planned)

**Delivery Layer** (`internal/delivery/http/`)
- HTTP interface adapters:
  - `handler/` - HTTP request/response handling
  - `server/server.go` - Router setup, middleware, and dependency wiring

**Infrastructure** (`pkg/`)
- Shared utilities and external concerns:
  - `config/config.go` - Environment-based configuration with validation
  - `database/mongodb.go` - MongoDB connection management
  - `logger/logger.go` - Structured logging setup

### Dependency Flow
- **Domain** ← **Use Cases** ← **Handlers** ← **Server**
- **Use Cases** depend on **Repository Interfaces** (not implementations)
- **Repository Implementations** are injected at server startup
- Clean dependency inversion: inner layers never depend on outer layers

### Key Design Patterns
- **Dependency Injection**: All dependencies wired in `server/server.go`
- **Interface Segregation**: Small, focused repository interfaces
- **Single Responsibility**: Each layer has one clear purpose
- **Repository Pattern**: Data access abstracted behind interfaces

## Configuration Management

### Environment Variables
Required variables (see `env.example`):
- `MONGODB_URI` - MongoDB connection string
- `DATABASE_NAME` - Database name
- `JWT_SECRET` - JWT signing secret
- `SERVER_PORT` - Server port (default: 8080)

Optional variables with defaults:
- `SERVER_READ_TIMEOUT`, `SERVER_WRITE_TIMEOUT`, `SERVER_IDLE_TIMEOUT`
- `LOG_LEVEL`, `LOG_FORMAT`
- `DB_MAX_CONNECTIONS`, `DB_CONNECT_TIMEOUT`, `DB_QUERY_TIMEOUT`

### Configuration Loading
- Automatic `.env` file loading via `godotenv`
- Environment variables override `.env` file values
- Configuration validation ensures required values are present
- Centralized configuration structure in `pkg/config/config.go`

## Database Architecture

### MongoDB Integration
- **Driver**: Official MongoDB Go driver (`go.mongodb.org/mongo-driver`)
- **Connection**: Single database instance with connection pooling
- **Repository Pattern**: Each domain has its own repository interface and implementation
- **Collections**: Employees, Restaurants, Reservations, Cities, Auth tokens

### Repository Implementation
- All repositories implement interfaces from `internal/repository/interfaces.go`
- MongoDB-specific implementations in `internal/repository/mongodb/`
- Context-based operations for timeout and cancellation support
- Proper error handling and logging for database operations

## Server Architecture

### HTTP Server Setup
- **Router**: Chi router with middleware stack
- **Middleware**: Request ID, logging, recovery, timeout, CORS
- **Graceful Shutdown**: 30-second timeout for clean shutdown
- **Health Checks**: Available at `/health` endpoint

### API Structure
- **Base Path**: `/api/v1/`
- **Swagger**: Available at `/swagger/index.html`
- **Metrics**: Prometheus metrics at `/metrics`
- **Versioned APIs**: Clear API versioning strategy

### Route Organization
- Routes grouped by domain (`/employees`, `/restaurants`, `/cities`, `/auth`)
- Nested routes for related resources (e.g., `/restaurants/{id}/rooms`)
- Development endpoints under `/dev/` prefix
- Special auth endpoints: `/auth/tg` and `/auth/tg/{requestId}` for admin bot integration

## Development Guidelines

### Adding New Features
1. **Define Domain Entity**: Create in `internal/domain/{entity}/entity.go`
2. **Add Repository Interface**: Update `internal/repository/interfaces.go`
3. **Implement Repository**: Create in `internal/repository/mongodb/{entity}.go`
4. **Create Use Case**: Add to `internal/usecase/{entity}/`
5. **Add HTTP Handler**: Create in `internal/delivery/http/handler/{entity}.go`
6. **Wire Dependencies**: Update dependency injection in `internal/delivery/http/server/server.go`
7. **Add Routes**: Register routes in the router setup
8. **Update Swagger**: Add API documentation annotations

### Code Standards
- Go 1.23+ required
- Follow existing dependency injection patterns
- Use context for all database operations
- Implement proper error handling with structured logging
- Write Swagger documentation for all endpoints
- Follow clean architecture layer boundaries strictly

### Testing Strategy
- Unit tests for each layer in isolation
- Mock repository interfaces for use case testing
- Integration tests for HTTP handlers
- Use table-driven tests where appropriate
- Test error scenarios and edge cases
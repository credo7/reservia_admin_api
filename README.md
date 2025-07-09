# Reservia API - Clean Architecture

A restaurant reservation system API built with Go, following Google-style clean architecture principles.

## 🏗️ Architecture Overview

This project follows **Clean Architecture** principles with clear separation of concerns:

```
cmd/
├── server/           # Application entry point
pkg/
├── config/          # Configuration management
├── database/        # Database connections
├── logger/          # Structured logging
internal/
├── domain/          # Business entities (User, Restaurant, Reservation)
├── repository/      # Data access interfaces
├── repository/mongodb/  # MongoDB implementations
├── usecase/         # Business logic layer
├── delivery/http/   # HTTP handlers and server
```

## 🚀 Features Migrated

### ✅ Completed
- **Domain Entities**: User, Restaurant, Reservation, Auth, City
- **Repository Pattern**: MongoDB implementations with interfaces
- **Use Cases**: Business logic layer with dependency injection
- **HTTP Handlers**: RESTful API endpoints with proper error handling
- **Clean Architecture**: Proper dependency inversion and separation of concerns
- **Configuration**: Environment-based configuration management
- **Logging**: Structured logging with logrus
- **Database**: MongoDB integration with proper connection handling

### 🔧 API Endpoints Available

#### User Management
- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users` - List users with pagination
- `GET /api/v1/users/{id}` - Get user by ID
- `PUT /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user

#### System
- `GET /health` - Health check endpoint
- `GET /api/v1/ping` - Development ping endpoint
- `GET /swagger/index.html` - **Swagger API Documentation** 📚

#### Restaurants
- `POST /api/v1/restaurants` - Create restaurant
- `GET /api/v1/restaurants` - List restaurants with filtering
- `GET /api/v1/restaurants/{id}` - Get restaurant by ID
- `PUT /api/v1/restaurants/{id}` - Update restaurant
- `DELETE /api/v1/restaurants/{id}` - Delete restaurant
- `GET /api/v1/restaurants/{id}/rooms` - Get restaurant rooms
- `GET /api/v1/restaurants/{id}/rooms/{roomId}` - Get specific room

## 📋 Requirements

- Go 1.21+
- MongoDB 4.4+
- Docker (optional)

## 🛠️ Installation & Setup

### 🚀 Quick Start (Recommended)

The easiest way to get started:

```bash
cd reservia-api-clean
./start.sh
```

This script will:
- Build the application
- Use your existing database environment
- Start the API server with all endpoints available

### 🔧 Manual Setup

### 1. Clone and Build

```bash
cd reservia-api-clean
go mod tidy
go build -o build/reservia-api cmd/server/main.go
```

### 2. Environment Configuration

Create a `.env` file or set environment variables:

```bash
# Required
MONGODB_URI=mongodb://localhost:27017
DATABASE_NAME=reservia_clean
JWT_SECRET=your-super-secret-jwt-key

# Optional (with defaults)
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=120s
```

### 3. Start MongoDB

```bash
# Using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Or using Docker Compose
docker-compose up -d mongodb
```

### 4. Run the Application

```bash
# Using environment variables
MONGODB_URI=mongodb://localhost:27017 \
DATABASE_NAME=reservia_clean \
JWT_SECRET=your-secret-key \
./build/reservia-api

# Or using .env file
./build/reservia-api
```

## 📚 API Documentation

### Swagger UI
Once the server is running, you can access the interactive API documentation at:
- **Swagger UI**: http://localhost:8080/swagger/index.html

The Swagger interface provides:
- 📖 Complete API documentation
- 🧪 Interactive endpoint testing
- 📝 Request/response examples
- 🔧 Schema definitions

### Generate Documentation
To regenerate the Swagger documentation after making changes:
```bash
make swagger
```

## 🧪 Testing the API

### Health Check
```bash
curl http://localhost:8080/health
```

### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "John Doe",
    "email": "john@example.com"
  }'
```

### List Users
```bash
curl http://localhost:8080/api/v1/users?limit=10&offset=0
```

### Get User by ID
```bash
curl http://localhost:8080/api/v1/users/{user_id}
```

## 🐳 Docker Support

### Build Docker Image
```bash
docker build -t reservia-api .
```

### Run with Docker Compose
```bash
docker-compose up -d
```

This starts:
- MongoDB
- Redis
- RabbitMQ
- Prometheus
- Grafana
- ELK Stack
- The API server

## 🔧 Development

### Project Structure
```
reservia-api-clean/
├── cmd/server/main.go           # Application entry point
├── pkg/                         # Shared packages
│   ├── config/                  # Configuration management
│   ├── database/                # Database connections
│   └── logger/                  # Structured logging
├── internal/                    # Private application code
│   ├── domain/                  # Business entities
│   │   ├── user/
│   │   ├── restaurant/
│   │   ├── reservation/
│   │   ├── auth/
│   │   └── city/
│   ├── repository/              # Data access interfaces
│   │   └── mongodb/             # MongoDB implementations
│   ├── usecase/                 # Business logic
│   └── delivery/http/           # HTTP layer
│       ├── handler/             # HTTP handlers
│       └── server/              # HTTP server
├── Makefile                     # Build automation
├── Dockerfile                   # Container image
└── docker-compose.dev.yml       # Development environment
```

### Adding New Features

1. **Add Domain Entity**: Create in `internal/domain/{entity}/`
2. **Add Repository Interface**: Update `internal/repository/interfaces.go`
3. **Add Repository Implementation**: Create in `internal/repository/mongodb/`
4. **Add Use Case**: Create in `internal/usecase/{entity}/`
5. **Add HTTP Handler**: Create in `internal/delivery/http/handler/`
6. **Wire Dependencies**: Update `internal/delivery/http/server/server.go`

### Build Commands

```bash
# Build the application
make build

# Run tests
make test

# Run with hot reload
make dev

# Build Docker image
make docker-build

# Deploy to development
make deploy-dev
```

## 📊 Monitoring

The application includes:
- Prometheus metrics at `/metrics`
- Structured logging with logrus
- Health checks at `/health`
- Grafana dashboards (when using Docker Compose)

## 🔄 Migration from Original

This clean architecture version provides:

### ✅ Improvements
- **Better Testability**: Easy to mock dependencies
- **Cleaner Code**: Separation of concerns
- **Scalability**: Easy to add new features
- **Maintainability**: Clear structure and dependencies
- **Professional Standards**: Following Google's Go style guide

### 🔧 Architectural Benefits
- **Dependency Inversion**: Use cases depend on interfaces, not implementations
- **Single Responsibility**: Each package has one clear purpose
- **Open/Closed Principle**: Easy to extend without modifying existing code
- **Interface Segregation**: Small, focused interfaces
- **Dependency Injection**: Clean wiring of components

## 📈 Next Steps

1. **Add Authentication Middleware**: JWT validation
2. **Add Restaurant Management**: CRUD operations
3. **Add Reservation System**: Booking logic
4. **Add Validation**: Request/response validation
5. **Add Testing**: Unit and integration tests
6. **Add Caching**: Redis integration
7. **Add Message Queue**: RabbitMQ for async operations

## 🤝 Contributing

1. Follow the existing architecture patterns
2. Add tests for new features
3. Update documentation
4. Follow Go best practices
5. Use dependency injection

## 📄 License

This project is licensed under the MIT License. 
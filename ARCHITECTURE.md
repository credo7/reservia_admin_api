# Clean Architecture Implementation

## 🔄 Before vs After: Code Quality Transformation

### Previous Structure (Problems)
```
reservia-api/
├── internal/
│   ├── handlers/          # Mixed concerns
│   ├── services/          # Business logic scattered
│   ├── models/            # Data models mixed with business logic
│   ├── middleware/        # Tightly coupled
│   ├── config/            # Configuration scattered
│   └── database/          # Database logic mixed with business logic
├── pkg/                   # Unclear separation
└── main.go               # Everything in one place
```

**Issues with Previous Structure:**
- ❌ **Tight Coupling**: Services directly depend on database implementations
- ❌ **Mixed Concerns**: Business logic mixed with infrastructure code
- ❌ **Poor Testability**: Hard to mock dependencies
- ❌ **Unclear Boundaries**: No clear separation between layers
- ❌ **Violation of SOLID**: Dependencies point inward instead of outward

### New Structure (Solutions)
```
reservia-api/
├── cmd/server/            # Application entry point
├── internal/              # Private application code
│   ├── domain/           # 🎯 Enterprise Business Rules
│   │   ├── user/         # User domain with entities & business rules
│   │   ├── restaurant/   # Restaurant domain with entities & business rules
│   │   ├── reservation/  # Reservation domain with entities & business rules
│   │   └── auth/         # Authentication domain
│   ├── repository/       # 🔌 Data Access Interfaces
│   │   ├── interfaces.go # Repository contracts
│   │   └── mongodb/      # MongoDB implementations
│   ├── usecase/          # 📋 Application Business Rules
│   │   ├── user/         # User use cases
│   │   ├── restaurant/   # Restaurant use cases
│   │   └── auth/         # Authentication use cases
│   └── delivery/         # 🚀 Interface Adapters
│       └── http/         # HTTP delivery layer
│           ├── handler/  # HTTP handlers
│           ├── middleware/ # HTTP middleware
│           ├── router/   # Route definitions
│           └── server/   # HTTP server setup
└── pkg/                  # 🔧 Public Libraries
    ├── auth/             # Authentication utilities
    ├── config/           # Configuration management
    ├── database/         # Database connection
    ├── logger/           # Structured logging
    └── validation/       # Input validation
```

## 🏗️ Clean Architecture Layers

### 1. **Domain Layer** (innermost)
- **Purpose**: Contains enterprise business rules and entities
- **Dependencies**: None (no external dependencies)
- **Examples**: User, Restaurant, Reservation entities with business logic

```go
// Example: User model with business rules
type User struct {
    ID    primitive.ObjectID
    Email string
    Role  Role
}

func (u *User) HasPermission(permission Permission) bool {
    // Business logic here
}
```

### 2. **Use Case Layer**
- **Purpose**: Contains application-specific business rules
- **Dependencies**: Only depends on domain layer and repository interfaces
- **Examples**: CreateUser, BookReservation, CancelReservation

```go
// Example: Use case depending on repository interface
type UserUseCase struct {
    userRepo repository.UserRepository
    logger   logger.Logger
}

func (uc *UserUseCase) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
    // Application business logic
}
```

### 3. **Interface Adapters Layer**
- **Purpose**: Converts data between use cases and external world
- **Dependencies**: Depends on use cases and domain
- **Examples**: HTTP handlers, repository implementations

```go
// Example: HTTP handler adapting web requests to use cases
type UserHandler struct {
    userUseCase usecase.UserUseCase
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Convert HTTP request to use case input
    // Call use case
    // Convert use case output to HTTP response
}
```

### 4. **Frameworks & Drivers Layer** (outermost)
- **Purpose**: Contains frameworks, databases, web servers
- **Dependencies**: Depends on all inner layers
- **Examples**: MongoDB driver, Chi router, Logrus logger

## 🎯 Key Improvements

### 1. **Dependency Inversion**
**Before:**
```go
// Service directly depends on database
type UserService struct {
    db *mongo.Database  // Concrete dependency
}
```

**After:**
```go
// Use case depends on interface
type UserUseCase struct {
    userRepo repository.UserRepository  // Interface dependency
}
```

### 2. **Single Responsibility**
**Before:**
```go
// Handler doing everything
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Validate input
    // Business logic
    // Database operations
    // Response formatting
}
```

**After:**
```go
// Each layer has single responsibility
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    req := h.parseRequest(r)           // Parsing
    user, err := h.userUseCase.Create(ctx, req)  // Business logic
    h.sendResponse(w, user, err)       // Response
}
```

### 3. **Testability**
**Before:**
```go
// Hard to test - tightly coupled
func TestUserService(t *testing.T) {
    // Need real database connection
    db := setupRealDatabase()
    service := NewUserService(db)
    // Test with real database
}
```

**After:**
```go
// Easy to test with mocks
func TestUserUseCase(t *testing.T) {
    mockRepo := &MockUserRepository{}
    useCase := NewUserUseCase(mockRepo)
    // Test with mock
}
```

### 4. **Configuration Management**
**Before:**
```go
// Configuration scattered throughout codebase
var jwtSecret = os.Getenv("JWT_SECRET")
var dbURL = os.Getenv("DATABASE_URL")
```

**After:**
```go
// Centralized configuration with validation
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Auth     AuthConfig
}

func Load() (*Config, error) {
    // Load and validate all configuration
}
```

### 5. **Error Handling**
**Before:**
```go
// Inconsistent error handling
if err != nil {
    log.Println("Error:", err)
    http.Error(w, "Internal error", 500)
}
```

**After:**
```go
// Structured error handling with context
type AppError struct {
    Code    string
    Message string
    Cause   error
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
    // Consistent error response formatting
}
```

## 📊 Benefits Achieved

### 1. **Maintainability**
- ✅ Clear separation of concerns
- ✅ Easy to locate and modify specific functionality
- ✅ Reduced cognitive load when reading code

### 2. **Testability**
- ✅ Easy to write unit tests for each layer
- ✅ Mock dependencies for isolated testing
- ✅ Test business logic without infrastructure

### 3. **Scalability**
- ✅ Easy to add new features without affecting existing code
- ✅ Multiple developers can work on different layers
- ✅ Clear interfaces for team collaboration

### 4. **Flexibility**
- ✅ Easy to swap implementations (e.g., MongoDB → PostgreSQL)
- ✅ Support multiple delivery mechanisms (HTTP, gRPC, CLI)
- ✅ Independent deployment of different layers

## 🔧 Development Workflow

### 1. **Adding New Feature**
1. Define domain entities and business rules
2. Create repository interface
3. Implement use case
4. Create HTTP handler
5. Add routes and middleware
6. Write tests for each layer

### 2. **Testing Strategy**
- **Unit Tests**: Test each layer in isolation
- **Integration Tests**: Test layer interactions
- **End-to-End Tests**: Test complete user workflows

### 3. **Deployment**
- **Local Development**: Use docker-compose.dev.yml
- **Staging**: Deploy with staging configurations
- **Production**: Use production-ready configurations

## 🚀 Getting Started with Clean Architecture

### 1. **Study the Structure**
```bash
# Explore the codebase
find . -name "*.go" | head -20
```

### 2. **Understand Dependencies**
```bash
# See dependency flow
go mod graph | grep github.com/reservia/api
```

### 3. **Run Tests**
```bash
# Test each layer
make test
make test-coverage
```

### 4. **Add New Feature**
1. Start with domain layer
2. Add repository interface
3. Implement use case
4. Create HTTP handler
5. Add tests

This clean architecture ensures your codebase remains maintainable, testable, and scalable as it grows. 
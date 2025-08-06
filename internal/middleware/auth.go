// Package middleware provides HTTP middleware for the admin API.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/service"
	"github.com/reservia/api/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthMiddleware handles authentication for admin API endpoints.
type AuthMiddleware struct {
	authService     *service.AuthService
	employeeService *service.EmployeeService
	logger          logger.Logger
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(authService *service.AuthService, employeeService *service.EmployeeService, logger logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService:     authService,
		employeeService: employeeService,
		logger:          logger,
	}
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// EmployeeIDKey is the context key for employee ID.
	EmployeeIDKey contextKey = "employee_id"
	// EmployeeKey is the context key for employee object.
	EmployeeKey contextKey = "employee"
)

// RequireAuth is a middleware that requires valid authentication.
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.writeError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		// Parse Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			am.writeError(w, http.StatusUnauthorized, "Invalid authorization header format. Expected: Bearer <token>")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			am.writeError(w, http.StatusUnauthorized, "Token is required")
			return
		}

		// Validate token and extract employee ID
		employeeID, err := am.authService.ValidateToken(token)
		if err != nil {
			am.logger.Error("Token validation failed", "error", err, "path", r.URL.Path)
			am.writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Get employee by ID
		employee, err := am.employeeService.GetEmployeeByID(r.Context(), employeeID)
		if err != nil {
			am.logger.Error("Failed to get employee during authentication", "employee_id", employeeID, "error", err)
			am.writeError(w, http.StatusUnauthorized, "Employee not found")
			return
		}

		// Add employee information to request context
		ctx := context.WithValue(r.Context(), EmployeeIDKey, employeeID)
		ctx = context.WithValue(ctx, EmployeeKey, employee)

		am.logger.Debug("Employee authenticated successfully", "employee_id", employeeID, "email", employee.Email, "path", r.URL.Path)

		// Continue to next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission is a middleware that requires specific permissions.
func (am *AuthMiddleware) RequirePermission(permission model.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get employee from context (should be set by RequireAuth middleware)
			employee, ok := GetEmployeeFromContext(r.Context())
			if !ok {
				am.logger.Error("Employee not found in context for permission check", "permission", permission)
				am.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Check if employee has the required permission
			if !employee.HasPermission(permission) {
				am.logger.Warn("Employee lacks required permission",
					"employee_id", employee.ID,
					"permission", permission,
					"path", r.URL.Path)
				am.writeError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			am.logger.Debug("Permission check passed", "employee_id", employee.ID, "permission", permission)

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRestaurantPermission is a middleware that requires specific permission for restaurant operations.
// The restaurant IDs should be extracted from the request (URL params, body, etc.) by the handler.
func (am *AuthMiddleware) RequireRestaurantPermission(permission model.Permission, restaurantIDExtractor func(*http.Request) []primitive.ObjectID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get employee from context
			employee, ok := GetEmployeeFromContext(r.Context())
			if !ok {
				am.logger.Error("Employee not found in context for restaurant access check")
				am.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Extract restaurant IDs from request
			restaurantIDs := restaurantIDExtractor(r)
			if len(restaurantIDs) == 0 {
				am.logger.Error("No restaurant IDs found in request")
				am.writeError(w, http.StatusBadRequest, "Restaurant ID required")
				return
			}

			// Check if employee has the required permission for at least one restaurant
			hasAccess := false
			for _, restaurantID := range restaurantIDs {
				// Check role-based permission for this specific restaurant
				role := employee.GetRoleInRestaurant(restaurantID)
				if role != nil {
					switch *role {
					case model.RoleOwner:
						hasAccess = true // Owner has all permissions
					case model.RoleAdmin:
						hasAccess = permission == model.PermissionReadReservations ||
							permission == model.PermissionUpdateReservations ||
							permission == model.PermissionCreateReservations ||
							permission == model.PermissionManageEmployees ||
							permission == model.PermissionManageRestaurants
					case model.RoleEmployee:
						hasAccess = permission == model.PermissionReadReservations ||
							permission == model.PermissionUpdateReservations ||
							permission == model.PermissionCreateReservations
					}
					if hasAccess {
						break
					}
				}
			}

			if !hasAccess {
				am.logger.Warn("Employee attempted to access restaurants without permission",
					"employee_id", employee.ID,
					"permission", permission,
					"restaurants", restaurantIDs,
					"path", r.URL.Path)
				am.writeError(w, http.StatusForbidden, "You don't have permission to perform this action on the specified restaurants")
				return
			}

			am.logger.Debug("Restaurant permission check passed", "employee_id", employee.ID, "permission", permission, "restaurants", restaurantIDs)

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRestaurantOwnerOrAdmin is a convenience middleware that requires owner or admin access to specific restaurants.
// This is equivalent to RequireRestaurantPermission with PermissionManageEmployees.
func (am *AuthMiddleware) RequireRestaurantOwnerOrAdmin(restaurantIDExtractor func(*http.Request) []primitive.ObjectID) func(http.Handler) http.Handler {
	return am.RequireRestaurantPermission(model.PermissionManageEmployees, restaurantIDExtractor)
}

// Helper functions to extract data from context

// GetEmployeeIDFromContext extracts the employee ID from the request context.
func GetEmployeeIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	employeeID, ok := ctx.Value(EmployeeIDKey).(primitive.ObjectID)
	return employeeID, ok
}

// GetEmployeeFromContext extracts the employee object from the request context.
func GetEmployeeFromContext(ctx context.Context) (*model.Employee, bool) {
	employee, ok := ctx.Value(EmployeeKey).(*model.Employee)
	return employee, ok
}

// Restaurant ID extraction helpers

// ExtractRestaurantIDFromURLParam creates an extractor function that gets restaurant ID from URL parameter.
func ExtractRestaurantIDFromURLParam(paramName string) func(*http.Request) []primitive.ObjectID {
	return func(r *http.Request) []primitive.ObjectID {
		idStr := chi.URLParam(r, paramName)
		if idStr == "" {
			return nil
		}
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return nil
		}
		return []primitive.ObjectID{id}
	}
}

// ExtractRestaurantIDsFromBody creates an extractor function that gets restaurant IDs from request body.
// The bodyField parameter specifies the JSON field name containing the restaurant IDs.
func ExtractRestaurantIDsFromBody(bodyField string) func(*http.Request) []primitive.ObjectID {
	return func(r *http.Request) []primitive.ObjectID {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil
		}

		// Reset body for subsequent reads
		r.Body = io.NopCloser(bytes.NewReader([]byte{}))

		idsInterface, ok := body[bodyField]
		if !ok {
			return nil
		}

		var ids []primitive.ObjectID
		switch v := idsInterface.(type) {
		case []interface{}:
			for _, idInterface := range v {
				if idStr, ok := idInterface.(string); ok {
					if id, err := primitive.ObjectIDFromHex(idStr); err == nil {
						ids = append(ids, id)
					}
				}
			}
		case string:
			if id, err := primitive.ObjectIDFromHex(v); err == nil {
				ids = append(ids, id)
			}
		}

		return ids
	}
}

// writeError writes a JSON error response.
func (am *AuthMiddleware) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		am.logger.Error("Failed to encode error response", "error", err)
	}
}

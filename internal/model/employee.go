// Package employee contains the employee model entities and logic.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Employee represents an employee in the system.
type Employee struct {
	ID               primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Email            string               `json:"email,omitempty" bson:"email,omitempty"`
	FullName         string               `json:"fullName" bson:"full_name"`
	TelegramIsBot    *bool                `json:"tgIsBot,omitempty" bson:"tg_is_bot,omitempty"`
	TelegramID       *int64               `json:"tgId,omitempty" bson:"tg_id,omitempty"`
	TelegramChatID   *int64               `json:"tgChatId,omitempty" bson:"tg_chat_id,omitempty"`
	TelegramUsername *string              `json:"tgUsername,omitempty" bson:"tg_username,omitempty"`
	TelegramLang     *string              `json:"tgLang,omitempty" bson:"tg_lang,omitempty"`
	TelegramPremium  *bool                `json:"tgIsPremium,omitempty" bson:"tg_is_premium,omitempty"`
	Restaurants      []EmployeeRestaurant `json:"restaurants" bson:"restaurants"`
	CreatedAt        time.Time            `json:"createdAt" bson:"created_at"`
	UpdatedAt        time.Time            `json:"updatedAt" bson:"updated_at"`
}

// EmployeeRestaurant represents an employee's role in a restaurant.
type EmployeeRestaurant struct {
	RestaurantID primitive.ObjectID `json:"restaurantId" bson:"restaurant_id"`
	Role         Role               `json:"role" bson:"role"`
}

// Role represents employee roles in the system.
type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleEmployee Role = "employee"
)

// IsValid checks if the role is valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleEmployee:
		return true
	default:
		return false
	}
}

// Permission represents a permission in the system.
type Permission string

const (
	PermissionCreateReservations  Permission = "create_reservations"
	PermissionReadReservations    Permission = "read_reservations"
	PermissionReadOwnReservations Permission = "read_own_reservations"
	PermissionUpdateReservations  Permission = "update_reservations"
	PermissionDeleteReservations  Permission = "delete_reservations"
	PermissionManageRestaurants   Permission = "manage_restaurants"
	PermissionManageEmployees     Permission = "manage_employees"
)

// HasPermission checks if the employee has a specific permission.
func (e *Employee) HasPermission(permission Permission) bool {
	// Check if employee has any restaurant roles
	for _, restaurant := range e.Restaurants {
		switch restaurant.Role {
		case RoleOwner:
			return true // Owner has all permissions
		case RoleAdmin:
			return permission == PermissionReadReservations ||
				permission == PermissionUpdateReservations ||
				permission == PermissionCreateReservations ||
				permission == PermissionManageEmployees ||
				permission == PermissionManageRestaurants
		case RoleEmployee:
			return permission == PermissionReadReservations ||
				permission == PermissionUpdateReservations ||
				permission == PermissionCreateReservations
		}
	}

	return false
}

// HasPermissionForRestaurant checks if the employee has a specific permission for a given restaurant.
func (e *Employee) HasPermissionForRestaurant(permission Permission, restaurantID primitive.ObjectID) bool {
	role := e.GetRoleInRestaurant(restaurantID)
	if role == nil {
		return false
	}

	switch *role {
	case RoleOwner:
		return true // Owner has all permissions
	case RoleAdmin:
		return permission == PermissionReadReservations ||
			permission == PermissionUpdateReservations ||
			permission == PermissionCreateReservations ||
			permission == PermissionManageEmployees ||
			permission == PermissionManageRestaurants
	case RoleEmployee:
		return permission == PermissionReadReservations ||
			permission == PermissionUpdateReservations ||
			permission == PermissionCreateReservations
	}

	return false
}

// HasPermissionForRestaurants checks if the employee has a specific permission for any of the given restaurants.
func (e *Employee) HasPermissionForRestaurants(permission Permission, restaurantIDs []primitive.ObjectID) bool {
	for _, restaurantID := range restaurantIDs {
		if e.HasPermissionForRestaurant(permission, restaurantID) {
			return true
		}
	}
	return false
}

// HasEmail checks if employee has email configured.
func (e *Employee) HasEmail() bool {
	return e.Email != ""
}

// HasTelegram checks if employee has Telegram configured.
func (e *Employee) HasTelegram() bool {
	return e.TelegramID != nil && e.TelegramChatID != nil
}

// HasRoleInRestaurant checks if employee has any role in the given restaurant.
func (e *Employee) HasRoleInRestaurant(restaurantID primitive.ObjectID) bool {
	for _, restaurant := range e.Restaurants {
		if restaurant.RestaurantID == restaurantID {
			return true
		}
	}
	return false
}

// GetRestaurantIDs returns a slice of restaurant IDs the employee has access to.
func (e *Employee) GetRestaurantIDs() []primitive.ObjectID {
	ids := make([]primitive.ObjectID, len(e.Restaurants))
	for i, restaurant := range e.Restaurants {
		ids[i] = restaurant.RestaurantID
	}
	return ids
}

// FilterRestaurantsByAccess returns a copy of the employee with only restaurants that the requester has access to.
// This prevents information leakage about restaurants the requester doesn't have access to.
func (e *Employee) FilterRestaurantsByAccess(requesterRestaurantIDs []primitive.ObjectID) *Employee {
	// Create a map for faster lookups
	requesterRestaurants := make(map[primitive.ObjectID]bool)
	for _, id := range requesterRestaurantIDs {
		requesterRestaurants[id] = true
	}

	// Filter restaurants to only include those the requester has access to
	var filteredRestaurants []EmployeeRestaurant
	for _, restaurant := range e.Restaurants {
		if requesterRestaurants[restaurant.RestaurantID] {
			filteredRestaurants = append(filteredRestaurants, restaurant)
		}
	}

	// Create a copy of the employee with filtered restaurants
	filteredEmployee := *e // Copy the struct
	filteredEmployee.Restaurants = filteredRestaurants
	return &filteredEmployee
}

// GetRoleInRestaurant returns the employee's role in a specific restaurant.
func (e *Employee) GetRoleInRestaurant(restaurantID primitive.ObjectID) *Role {
	for _, restaurant := range e.Restaurants {
		if restaurant.RestaurantID == restaurantID {
			return &restaurant.Role
		}
	}
	return nil
}

// IsOwnerOfRestaurant checks if employee is owner of the given restaurant.
func (e *Employee) IsOwnerOfRestaurant(restaurantID primitive.ObjectID) bool {
	role := e.GetRoleInRestaurant(restaurantID)
	return role != nil && *role == RoleOwner
}

// IsAdminOfRestaurant checks if employee is admin of the given restaurant.
func (e *Employee) IsAdminOfRestaurant(restaurantID primitive.ObjectID) bool {
	role := e.GetRoleInRestaurant(restaurantID)
	return role != nil && (*role == RoleOwner || *role == RoleAdmin)
}

// CreateEmployeeRequest represents a request to create a new employee.
type CreateEmployeeRequest struct {
	Email            string               `json:"email,omitempty" validate:"omitempty,email"`
	FullName         string               `json:"fullName" validate:"required,min=2,max=100"`
	Role             string               `json:"role,omitempty"`
	TelegramIsBot    *bool                `json:"tgIsBot,omitempty"`
	TelegramID       *int64               `json:"tgId,omitempty"`
	TelegramChatID   *int64               `json:"tgChatId,omitempty"`
	TelegramUsername *string              `json:"tgUsername,omitempty"`
	TelegramLang     *string              `json:"tgLang,omitempty"`
	TelegramPremium  *bool                `json:"tgIsPremium,omitempty"`
	Restaurants      []EmployeeRestaurant `json:"restaurants,omitempty"`
}

// AdminUpdateEmployeeRequest represents a request to update an employee's role assignments (admin only).
// This matches the Python PutEmployeeSchema which only allows role management.
type AdminUpdateEmployeeRequest struct {
	RestaurantID primitive.ObjectID `json:"restaurantId" validate:"required"`
	Role         Role               `json:"role" validate:"required"`
}

// UpdateEmployeeRequest represents a request to update an employee's personal information.
// This is used internally by the service layer and should not be exposed to admin endpoints.
type UpdateEmployeeRequest struct {
	Email            *string `json:"email,omitempty" validate:"omitempty,email"`
	FullName         *string `json:"fullName,omitempty" validate:"omitempty,min=2,max=100"`
	TelegramIsBot    *bool   `json:"tgIsBot,omitempty"`
	TelegramID       *int64  `json:"tgId,omitempty"`
	TelegramChatID   *int64  `json:"tgChatId,omitempty"`
	TelegramUsername *string `json:"tgUsername,omitempty"`
	TelegramLang     *string `json:"tgLang,omitempty"`
	TelegramPremium  *bool   `json:"tgIsPremium,omitempty"`
}

// PersonalCabinetUpdateRequest represents the request to update personal cabinet info.
type PersonalCabinetUpdateRequest struct {
	FullName *string `json:"fullName,omitempty" validate:"omitempty,min=2,max=100"`
}

// Note: EmailUpdateRequest and EmailUpdateResponse are defined in auth.go

// TelegramConnectionResponse represents the response for Telegram connection.
type TelegramConnectionResponse struct {
	ConnectionRequestID string `json:"connectionRequestId"`
	TelegramURL         string `json:"tgUrl"`
}

// TelegramConnectionVerifyResponse represents the verification response for Telegram connection.
type TelegramConnectionVerifyResponse struct {
	IsPending bool `json:"isPending"`
	Success   bool `json:"success"`
}

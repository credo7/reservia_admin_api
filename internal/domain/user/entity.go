// Package user contains the user domain entities and logic.
package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system.
type User struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email            string             `json:"email,omitempty" bson:"email,omitempty"`
	FullName         string             `json:"full_name" bson:"full_name"`
	TelegramIsBot    *bool              `json:"tg_is_bot,omitempty" bson:"tg_is_bot,omitempty"`
	TelegramID       *int64             `json:"tg_id,omitempty" bson:"tg_id,omitempty"`
	TelegramChatID   *int64             `json:"tg_chat_id,omitempty" bson:"tg_chat_id,omitempty"`
	TelegramUsername *string            `json:"tg_username,omitempty" bson:"tg_username,omitempty"`
	TelegramLang     *string            `json:"tg_lang,omitempty" bson:"tg_lang,omitempty"`
	TelegramPremium  *bool              `json:"tg_is_premium,omitempty" bson:"tg_is_premium,omitempty"`
	Restaurants      []UserRestaurant   `json:"restaurants" bson:"restaurants"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
}

// UserRestaurant represents a user's role in a restaurant.
type UserRestaurant struct {
	RestaurantID primitive.ObjectID `json:"restaurant_id" bson:"restaurant_id"`
	Role         Role               `json:"role" bson:"role"`
}

// Role represents user roles in the system.
type Role string

const (
	RoleOwner    Role = "owner"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
	RoleUser     Role = "user"
)

// IsValid checks if the role is valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleManager, RoleEmployee, RoleUser:
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
	PermissionManageUsers         Permission = "manage_users"
)

// HasPermission checks if the user has a specific permission.
func (u *User) HasPermission(permission Permission) bool {
	// Check if user has any restaurant roles
	for _, restaurant := range u.Restaurants {
		switch restaurant.Role {
		case RoleOwner:
			return true // Owner has all permissions
		case RoleManager:
			return permission == PermissionReadReservations ||
				permission == PermissionUpdateReservations ||
				permission == PermissionCreateReservations
		case RoleEmployee:
			return permission == PermissionReadReservations ||
				permission == PermissionUpdateReservations
		}
	}

	// Regular users can only create reservations and read their own
	return permission == PermissionCreateReservations ||
		permission == PermissionReadOwnReservations
}

// HasEmail checks if user has email configured.
func (u *User) HasEmail() bool {
	return u.Email != ""
}

// HasTelegram checks if user has Telegram configured.
func (u *User) HasTelegram() bool {
	return u.TelegramID != nil && u.TelegramChatID != nil
}

// HasRoleInRestaurant checks if user has any role in the given restaurant.
func (u *User) HasRoleInRestaurant(restaurantID primitive.ObjectID) bool {
	for _, restaurant := range u.Restaurants {
		if restaurant.RestaurantID == restaurantID {
			return true
		}
	}
	return false
}

// GetRestaurantIDs returns a slice of restaurant IDs the user has access to.
func (u *User) GetRestaurantIDs() []primitive.ObjectID {
	ids := make([]primitive.ObjectID, len(u.Restaurants))
	for i, restaurant := range u.Restaurants {
		ids[i] = restaurant.RestaurantID
	}
	return ids
}

// GetRoleInRestaurant returns the user's role in a specific restaurant.
func (u *User) GetRoleInRestaurant(restaurantID primitive.ObjectID) *Role {
	for _, restaurant := range u.Restaurants {
		if restaurant.RestaurantID == restaurantID {
			return &restaurant.Role
		}
	}
	return nil
}

// IsOwnerOfRestaurant checks if user is owner of the given restaurant.
func (u *User) IsOwnerOfRestaurant(restaurantID primitive.ObjectID) bool {
	role := u.GetRoleInRestaurant(restaurantID)
	return role != nil && *role == RoleOwner
}

// IsManagerOfRestaurant checks if user is manager of the given restaurant.
func (u *User) IsManagerOfRestaurant(restaurantID primitive.ObjectID) bool {
	role := u.GetRoleInRestaurant(restaurantID)
	return role != nil && (*role == RoleOwner || *role == RoleManager)
}

// CreateUserRequest represents a request to create a new user.
type CreateUserRequest struct {
	Email            string           `json:"email,omitempty" validate:"omitempty,email"`
	FullName         string           `json:"full_name" validate:"required,min=2,max=100"`
	Role             string           `json:"role,omitempty"`
	TelegramIsBot    *bool            `json:"tg_is_bot,omitempty"`
	TelegramID       *int64           `json:"tg_id,omitempty"`
	TelegramChatID   *int64           `json:"tg_chat_id,omitempty"`
	TelegramUsername *string          `json:"tg_username,omitempty"`
	TelegramLang     *string          `json:"tg_lang,omitempty"`
	TelegramPremium  *bool            `json:"tg_is_premium,omitempty"`
	Restaurants      []UserRestaurant `json:"restaurants,omitempty"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Email            *string `json:"email,omitempty" validate:"omitempty,email"`
	FullName         *string `json:"full_name,omitempty" validate:"omitempty,min=2,max=100"`
	TelegramIsBot    *bool   `json:"tg_is_bot,omitempty"`
	TelegramID       *int64  `json:"tg_id,omitempty"`
	TelegramChatID   *int64  `json:"tg_chat_id,omitempty"`
	TelegramUsername *string `json:"tg_username,omitempty"`
	TelegramLang     *string `json:"tg_lang,omitempty"`
	TelegramPremium  *bool   `json:"tg_is_premium,omitempty"`
}

// PersonalCabinetUpdateRequest represents the request to update personal cabinet info.
type PersonalCabinetUpdateRequest struct {
	FullName *string `json:"full_name,omitempty" validate:"omitempty,min=2,max=100"`
}

// EmailUpdateRequest represents the request to update email.
type EmailUpdateRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
}

// EmailUpdateResponse represents the response to email update request.
type EmailUpdateResponse struct {
	CodeRequestID string `json:"code_request_id"`
	Message       string `json:"message"`
}

// TelegramConnectionResponse represents the response for Telegram connection.
type TelegramConnectionResponse struct {
	ConnectionRequestID string `json:"connection_request_id"`
	TelegramURL         string `json:"tg_url"`
}

// TelegramConnectionVerifyResponse represents the verification response for Telegram connection.
type TelegramConnectionVerifyResponse struct {
	IsPending bool `json:"is_pending"`
	Success   bool `json:"success"`
}

// UserResponse represents the response when returning user data.
type UserResponse struct {
	ID               primitive.ObjectID `json:"id"`
	Email            string             `json:"email,omitempty"`
	FullName         string             `json:"full_name"`
	TelegramIsBot    *bool              `json:"tg_is_bot,omitempty"`
	TelegramID       *int64             `json:"tg_id,omitempty"`
	TelegramChatID   *int64             `json:"tg_chat_id,omitempty"`
	TelegramUsername *string            `json:"tg_username,omitempty"`
	TelegramLang     *string            `json:"tg_lang,omitempty"`
	TelegramPremium  *bool              `json:"tg_is_premium,omitempty"`
	Restaurants      []UserRestaurant   `json:"restaurants"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// ToResponse converts a User to UserResponse.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		FullName:         u.FullName,
		TelegramIsBot:    u.TelegramIsBot,
		TelegramID:       u.TelegramID,
		TelegramChatID:   u.TelegramChatID,
		TelegramUsername: u.TelegramUsername,
		TelegramLang:     u.TelegramLang,
		TelegramPremium:  u.TelegramPremium,
		Restaurants:      u.Restaurants,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

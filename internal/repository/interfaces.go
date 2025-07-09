// Package repository defines the repository interfaces for data access.
package repository

import (
	"context"

	"github.com/reservia/api/internal/domain/auth"
	"github.com/reservia/api/internal/domain/city"
	"github.com/reservia/api/internal/domain/reservation"
	"github.com/reservia/api/internal/domain/restaurant"
	"github.com/reservia/api/internal/domain/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *user.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*user.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*user.User, error)

	// GetByTelegramID retrieves a user by telegram ID
	GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error)

	// Update updates a user
	Update(ctx context.Context, user *user.User) error

	// Delete deletes a user
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves users with pagination
	List(ctx context.Context, limit, offset int) ([]*user.User, error)
}

// RestaurantRepository defines the interface for restaurant data access.
type RestaurantRepository interface {
	// Create creates a new restaurant
	Create(ctx context.Context, restaurant *restaurant.Restaurant) error

	// GetByID retrieves a restaurant by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*restaurant.Restaurant, error)

	// Update updates a restaurant
	Update(ctx context.Context, restaurant *restaurant.Restaurant) error

	// Delete deletes a restaurant
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves restaurants with filtering and pagination
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*restaurant.Restaurant, error)

	// GetByLocation retrieves restaurants by location
	GetByLocation(ctx context.Context, latitude, longitude, radius float64) ([]*restaurant.Restaurant, error)
}

// ReservationRepository defines the interface for reservation data access.
type ReservationRepository interface {
	// Create creates a new reservation
	Create(ctx context.Context, reservation *reservation.Reservation) error

	// GetByID retrieves a reservation by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*reservation.Reservation, error)

	// Update updates a reservation
	Update(ctx context.Context, reservation *reservation.Reservation) error

	// Delete deletes a reservation
	Delete(ctx context.Context, id primitive.ObjectID) error

	// GetByUserID retrieves reservations by user ID
	GetByUserID(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*reservation.Reservation, error)

	// GetByRestaurantID retrieves reservations by restaurant ID
	GetByRestaurantID(ctx context.Context, restaurantID primitive.ObjectID, limit, offset int) ([]*reservation.Reservation, error)

	// GetByDateRange retrieves reservations within a date range
	GetByDateRange(ctx context.Context, restaurantID primitive.ObjectID, startDate, endDate string) ([]*reservation.Reservation, error)

	// CheckAvailability checks if a time slot is available
	CheckAvailability(ctx context.Context, restaurantID, roomID primitive.ObjectID, startTime, endTime string) (bool, error)
}

// CityRepository defines the interface for city data access.
type CityRepository interface {
	// GetAll retrieves all cities
	GetAll(ctx context.Context) ([]*city.City, error)

	// GetByName retrieves a city by name
	GetByName(ctx context.Context, name string) (*city.City, error)

	// GetByCountry retrieves cities by country
	GetByCountry(ctx context.Context, country string) ([]*city.City, error)
}

// AuthRepository defines the interface for authentication data access.
type AuthRepository interface {
	// CreateVerificationCode creates a verification code
	CreateVerificationCode(ctx context.Context, code *auth.VerificationCode) error

	// GetVerificationCode retrieves a verification code
	GetVerificationCode(ctx context.Context, email, code string) (*auth.VerificationCode, error)

	// DeleteVerificationCode deletes a verification code
	DeleteVerificationCode(ctx context.Context, email, code string) error

	// CreateRefreshToken creates a refresh token
	CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error

	// GetRefreshToken retrieves a refresh token
	GetRefreshToken(ctx context.Context, token string) (*auth.RefreshToken, error)

	// DeleteRefreshToken deletes a refresh token
	DeleteRefreshToken(ctx context.Context, token string) error
}

// Repositories aggregates all repository interfaces.
type Repositories struct {
	User        UserRepository
	Restaurant  RestaurantRepository
	Reservation ReservationRepository
	City        CityRepository
	Auth        AuthRepository
}

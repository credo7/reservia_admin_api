// Package repository defines the repository interfaces for data access.
package repository

import (
	"context"
	"time"
	"github.com/reservia/api/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmployeeRepository defines the interface for employee data access.
type EmployeeRepository interface {
	// Create creates a new employee
	Create(ctx context.Context, employee *model.Employee) error

	// GetByID retrieves an employee by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Employee, error)

	// GetByEmail retrieves an employee by email
	GetByEmail(ctx context.Context, email string) (*model.Employee, error)

	// GetByTelegramID retrieves an employee by telegram ID
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.Employee, error)

	// Update updates an employee
	Update(ctx context.Context, employee *model.Employee) error

	// Delete deletes an employee
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves employees with pagination
	List(ctx context.Context, limit, offset int) ([]*model.Employee, error)

	// ListByRestaurantIDs retrieves employees who have access to any of the specified restaurants
	ListByRestaurantIDs(ctx context.Context, restaurantIDs []primitive.ObjectID) ([]*model.Employee, error)
}

// RestaurantRepository defines the interface for restaurant data access.
type RestaurantRepository interface {
	// Create creates a new restaurant
	Create(ctx context.Context, restaurant *model.Restaurant) error

	// GetByID retrieves a restaurant by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Restaurant, error)

	// GetByURLName retrieves a restaurant by URL name
	GetByURLName(ctx context.Context, urlName string) (*model.Restaurant, error)

	// Update updates a restaurant
	Update(ctx context.Context, restaurant *model.Restaurant) error

	// Delete deletes a restaurant
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves restaurants with filtering and pagination
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*model.Restaurant, error)

	// GetByLocation retrieves restaurants by location
	GetByLocation(ctx context.Context, latitude, longitude, radius float64) ([]*model.Restaurant, error)

	// GetByIDs retrieves restaurants by a list of IDs
	GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*model.Restaurant, error)
}

// ReservationRepository defines the interface for reservation data access.
type ReservationRepository interface {
	// Create creates a new reservation
	Create(ctx context.Context, reservation *model.Reservation) error

	// GetByID retrieves a reservation by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Reservation, error)

	// Update updates a reservation
	Update(ctx context.Context, reservation *model.Reservation) error

	// Delete deletes a reservation
	Delete(ctx context.Context, id primitive.ObjectID) error

	// GetByRestaurantID retrieves reservations by restaurant ID
	GetByRestaurantID(ctx context.Context, restaurantID primitive.ObjectID, limit, offset int) ([]*model.Reservation, error)

	// GetByDateRange retrieves reservations within a date range
	GetByDateRange(ctx context.Context, restaurantID primitive.ObjectID, startDate, endDate string) ([]*model.Reservation, error)

	// CheckAvailability checks if a time slot is available
	CheckAvailability(ctx context.Context, restaurantID, roomID primitive.ObjectID, startTime, endTime string) (bool, error)

	// Enhanced availability methods (matching Python functionality)

	// GetActiveReservationsInTimeRange retrieves active reservations (PENDING, CONFIRMED, ARRIVED) that overlap with the given time range
	GetActiveReservationsInTimeRange(ctx context.Context, filters model.TableAvailabilityFilters) ([]*model.Reservation, error)

	// GetActiveReservationsForTable retrieves active reservations for a specific table in a time range  
	GetActiveReservationsForTable(ctx context.Context, restaurantID, tableID primitive.ObjectID, startAt, endAt time.Time) ([]*model.Reservation, error)

	// GetActiveReservationsForRoom retrieves active reservations for all tables in a room within a time range
	GetActiveReservationsForRoom(ctx context.Context, restaurantID, roomID primitive.ObjectID, startAt, endAt time.Time) ([]*model.Reservation, error)

	// GetActiveReservationsForMultipleTables retrieves active reservations for multiple tables in a time range
	GetActiveReservationsForMultipleTables(ctx context.Context, restaurantID primitive.ObjectID, tableIDs []primitive.ObjectID, startAt, endAt time.Time) ([]*model.Reservation, error)
}

// CityRepository defines the interface for city data access.
type CityRepository interface {
	// GetAll retrieves all cities
	GetAll(ctx context.Context) ([]*model.City, error)

	// GetByID retrieves a city by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.City, error)

	// GetByName retrieves a city by name
	GetByName(ctx context.Context, name string) (*model.City, error)

	// GetByCountry retrieves cities by country
	GetByCountry(ctx context.Context, country string) ([]*model.City, error)
}

// AuthRepository defines the interface for authentication data access.
type AuthRepository interface {
	// Login verification codes
	CreateLoginVerificationCode(ctx context.Context, code *model.LoginVerificationCode) error
	GetLoginVerificationCode(ctx context.Context, email, code string) (*model.LoginVerificationCode, error)
	GetLoginVerificationCodeByID(ctx context.Context, id primitive.ObjectID) (*model.LoginVerificationCode, error)
	UpdateLoginVerificationCode(ctx context.Context, code *model.LoginVerificationCode) error
	DeleteLoginVerificationCode(ctx context.Context, email, code string) error

	// Register verification codes
	CreateRegisterVerificationCode(ctx context.Context, code *model.RegisterVerificationCode) error
	GetRegisterVerificationCode(ctx context.Context, email, code string) (*model.RegisterVerificationCode, error)
	GetRegisterVerificationCodeByID(ctx context.Context, id primitive.ObjectID) (*model.RegisterVerificationCode, error)
	UpdateRegisterVerificationCode(ctx context.Context, code *model.RegisterVerificationCode) error
	DeleteRegisterVerificationCode(ctx context.Context, email, code string) error

	// Employee registration codes (for invitations)
	CreateEmployeeRegistrationCode(ctx context.Context, code *model.EmployeeRegistrationCode) error
	GetEmployeeRegistrationCode(ctx context.Context, requestID string) (*model.EmployeeRegistrationCode, error)
	UpdateEmployeeRegistrationCode(ctx context.Context, code *model.EmployeeRegistrationCode) error
	GetPendingEmployeeRegistrationCodes(ctx context.Context) ([]*model.EmployeeRegistrationCode, error)
	DeleteEmployeeRegistrationCode(ctx context.Context, requestID string) error

	// Telegram verification codes (for admin bot auth)
	CreateTelegramVerificationCode(ctx context.Context, code *model.TelegramVerificationCode) error
	GetTelegramVerificationCode(ctx context.Context, requestID string) (*model.TelegramVerificationCode, error)
	UpdateTelegramVerificationCode(ctx context.Context, code *model.TelegramVerificationCode) error
}

// Repositories aggregates all repository interfaces.
type Repositories struct {
	Employee    EmployeeRepository
	Restaurant  RestaurantRepository
	Reservation ReservationRepository
	City        CityRepository
	Auth        AuthRepository
}

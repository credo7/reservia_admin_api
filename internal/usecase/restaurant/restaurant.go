// Package restaurant provides restaurant-related use cases.
package restaurant

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/domain/restaurant"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
)

// UseCase handles restaurant-related business logic.
type UseCase struct {
	restaurantRepo repository.RestaurantRepository
	logger         logger.Logger
}

// New creates a new restaurant use case.
func New(restaurantRepo repository.RestaurantRepository, logger logger.Logger) *UseCase {
	return &UseCase{
		restaurantRepo: restaurantRepo,
		logger:         logger,
	}
}

// CreateRestaurant creates a new restaurant.
func (uc *UseCase) CreateRestaurant(ctx context.Context, req *restaurant.CreateRestaurantRequest) (*restaurant.Restaurant, error) {
	// Create restaurant
	newRestaurant := &restaurant.Restaurant{
		Name:        req.Name,
		URLName:     req.URLName,
		Address:     req.Address,
		Phone:       req.Phone,
		Email:       req.Email,
		Description: req.Description,
		IsActive:    true,
		UTCOffset:   req.UTCOffset,
		Settings:    req.Settings,
		Rooms:       req.Rooms,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.restaurantRepo.Create(ctx, newRestaurant); err != nil {
		return nil, fmt.Errorf("failed to create restaurant: %w", err)
	}

	uc.logger.Info("Restaurant created successfully", "restaurant_id", newRestaurant.ID.Hex())
	return newRestaurant, nil
}

// GetRestaurantByID retrieves a restaurant by ID.
func (uc *UseCase) GetRestaurantByID(ctx context.Context, id primitive.ObjectID) (*restaurant.Restaurant, error) {
	rest, err := uc.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant by ID: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}
	return rest, nil
}

// UpdateRestaurant updates a restaurant.
func (uc *UseCase) UpdateRestaurant(ctx context.Context, id primitive.ObjectID, req *restaurant.UpdateRestaurantRequest) (*restaurant.Restaurant, error) {
	// Get existing restaurant
	existingRestaurant, err := uc.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Update fields
	if req.Name != nil {
		existingRestaurant.Name = *req.Name
	}
	if req.Address != nil {
		existingRestaurant.Address = *req.Address
	}
	if req.Phone != nil {
		existingRestaurant.Phone = *req.Phone
	}
	if req.Email != nil {
		existingRestaurant.Email = *req.Email
	}
	if req.Description != nil {
		existingRestaurant.Description = *req.Description
	}
	if req.IsActive != nil {
		existingRestaurant.IsActive = *req.IsActive
	}
	if req.UTCOffset != nil {
		existingRestaurant.UTCOffset = *req.UTCOffset
	}
	if req.Settings != nil {
		existingRestaurant.Settings = *req.Settings
	}
	if req.Rooms != nil {
		existingRestaurant.Rooms = req.Rooms
	}

	existingRestaurant.UpdatedAt = time.Now()

	if err := uc.restaurantRepo.Update(ctx, existingRestaurant); err != nil {
		return nil, fmt.Errorf("failed to update restaurant: %w", err)
	}

	uc.logger.Info("Restaurant updated successfully", "restaurant_id", existingRestaurant.ID.Hex())
	return existingRestaurant, nil
}

// DeleteRestaurant deletes a restaurant.
func (uc *UseCase) DeleteRestaurant(ctx context.Context, id primitive.ObjectID) error {
	// Check if restaurant exists
	existingRestaurant, err := uc.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return fmt.Errorf("restaurant not found")
	}

	if err := uc.restaurantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete restaurant: %w", err)
	}

	uc.logger.Info("Restaurant deleted successfully", "restaurant_id", id.Hex())
	return nil
}

// ListRestaurants retrieves restaurants with filtering and pagination.
func (uc *UseCase) ListRestaurants(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*restaurant.Restaurant, error) {
	restaurants, err := uc.restaurantRepo.List(ctx, filters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list restaurants: %w", err)
	}
	return restaurants, nil
}

// GetRestaurantsByLocation retrieves restaurants by location.
func (uc *UseCase) GetRestaurantsByLocation(ctx context.Context, latitude, longitude, radius float64) ([]*restaurant.Restaurant, error) {
	restaurants, err := uc.restaurantRepo.GetByLocation(ctx, latitude, longitude, radius)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurants by location: %w", err)
	}
	return restaurants, nil
}

// GetRestaurantRoom retrieves a specific room from a restaurant.
func (uc *UseCase) GetRestaurantRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID string) (*restaurant.Room, error) {
	rest, err := uc.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	room := rest.GetRoom(roomID)
	if room == nil {
		return nil, fmt.Errorf("room not found")
	}

	return room, nil
}

// GetRestaurantRooms retrieves all rooms for a restaurant.
func (uc *UseCase) GetRestaurantRooms(ctx context.Context, restaurantID primitive.ObjectID) ([]restaurant.Room, error) {
	rest, err := uc.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	return rest.Rooms, nil
}

// GetActiveRestaurantRooms retrieves all active rooms for a restaurant.
func (uc *UseCase) GetActiveRestaurantRooms(ctx context.Context, restaurantID primitive.ObjectID) ([]restaurant.Room, error) {
	rest, err := uc.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	return rest.GetActiveRooms(), nil
}

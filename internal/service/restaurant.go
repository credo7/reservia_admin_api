// Package service provides restaurant-related services.
package service

import (
	"context"
	"fmt"
	"github.com/reservia/api/internal/model"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
)

// Service handles restaurant-related business logic.
type RestaurantService struct {
	restaurantRepo repository.RestaurantRepository
	employeeRepo   repository.EmployeeRepository
	cityService    *CityService
	logger         logger.Logger
}

// NewRestaurantService creates a new restaurant service.
func NewRestaurantService(restaurantRepo repository.RestaurantRepository, employeeRepo repository.EmployeeRepository, cityService *CityService, logger logger.Logger) *RestaurantService {
	return &RestaurantService{
		restaurantRepo: restaurantRepo,
		employeeRepo:   employeeRepo,
		cityService:    cityService,
		logger:         logger,
	}
}

// CreateRestaurant creates a new restaurant with bootstrap logic (matches Python implementation).
// The creating employee automatically becomes the owner of the restaurant.
func (rs *RestaurantService) CreateRestaurant(ctx context.Context, creator *model.Employee, req *model.CreateRestaurantRequest) (*model.Restaurant, error) {
	// 1. Look up city to get UTC offset (using city ID)
	city, err := rs.cityService.GetCityByID(ctx, req.CityID)
	if err != nil {
		rs.logger.Error("Failed to find city", "cityId", req.CityID.Hex(), "error", err)
		return nil, fmt.Errorf("city not found: %s", req.CityID.Hex())
	}

	if city == nil {
		rs.logger.Error("City not found", "cityId", req.CityID.Hex())
		return nil, fmt.Errorf("city not found: %s", req.CityID.Hex())
	}

	// 2. Generate URL name from restaurant name (matching Python slugification)
	urlName := model.GenerateURLName(req.Name)

	// 3. Check if URL name is available and make it unique if needed
	baseURLName := urlName
	counter := 1
	for {
		existing, err := rs.restaurantRepo.GetByURLName(ctx, urlName)
		if err != nil {
			return nil, fmt.Errorf("failed to check URL name availability: %w", err)
		}
		if existing == nil {
			break // URL name is available
		}
		// Make it unique by appending a number
		urlName = fmt.Sprintf("%s-%d", baseURLName, counter)
		counter++
		if counter > 100 { // Prevent infinite loop
			return nil, fmt.Errorf("could not generate unique URL name for restaurant")
		}
	}

	// 4. Apply default settings (matching Python RestaurantSettingsSchema defaults)
	defaultSettings := model.Settings{
		AutoReservationAccepting:   true,
		ReservationIntervalMinutes: 30,     // Python default: 30 minutes
		ReservationReminderMinutes: 15,     // Python default: 15 minutes
		MinReservationDuration:     "1:00", // Python default: "1:00" (HH:MM format)
		MaxReservationDuration:     "3:00", // Python default: "3:00" (HH:MM format)
		MaxDaysInAdvance:           31,     // Python default: 31 days
		// Legacy fields (for backward compatibility, will be deprecated)
		MaxGuestsPerReservation:    8,   // Legacy default
		DefaultReservationDuration: 120, // Legacy default: 2 hours in minutes
	}

	// 5. Create restaurant with minimal required data, server-generated values, and defaults
	newRestaurant := &model.Restaurant{
		Name:        req.Name,
		URLName:     urlName, // Auto-generated from name
		City:        city.Name,        // Auto-resolved from city
		Address:     req.Address,
		Phone:       req.Phone,
		Email:       "",               // Empty by default, can be set later
		Description: "",               // Empty by default, can be set later
		IsActive:    false,            // Match Python: start disabled, requires manual activation
		UTCOffset:   city.UTCOffset,   // Auto-resolved from city
		Settings:    defaultSettings,  // Server-side defaults
		Rooms:       []model.Room{},   // Empty by default, rooms added later
		SubURLs:     []model.SubURL{}, // Empty by default
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 6. Create restaurant in database first
	if err := rs.restaurantRepo.Create(ctx, newRestaurant); err != nil {
		return nil, fmt.Errorf("failed to create restaurant: %w", err)
	}

	// 7. CRITICAL BOOTSTRAP STEP: Make the creating employee the owner of this restaurant
	// This matches the Python logic in RestaurantService.add()
	creatorCopy := *creator
	creatorCopy.Restaurants = append(creatorCopy.Restaurants, model.EmployeeRestaurant{
		RestaurantID: newRestaurant.ID,
		Role:         model.RoleOwner,
	})
	creatorCopy.UpdatedAt = time.Now()

	// 8. Update the employee with their new restaurant ownership
	if err := rs.employeeRepo.Update(ctx, &creatorCopy); err != nil {
		rs.logger.Error("Failed to establish restaurant ownership",
			"error", err, "employee_id", creator.ID, "restaurant_id", newRestaurant.ID)
		// This is critical - if we can't establish ownership, the restaurant is unusable
		return nil, fmt.Errorf("failed to establish restaurant ownership: %w", err)
	}

	rs.logger.Info("Restaurant created successfully with owner assignment",
		"restaurant_id", newRestaurant.ID.Hex(),
		"owner_employee_id", creator.ID.Hex(),
		"restaurant_name", newRestaurant.Name,
		"url_name", newRestaurant.URLName,
		"city", city.Name,
		"utc_offset", newRestaurant.UTCOffset)
	return newRestaurant, nil
}

// GetRestaurantByID retrieves a restaurant by ID.
func (rs *RestaurantService) GetRestaurantByID(ctx context.Context, id primitive.ObjectID) (*model.Restaurant, error) {
	rest, err := rs.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant by ID: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}
	return rest, nil
}

// UpdateRestaurant updates a restaurant.
func (rs *RestaurantService) UpdateRestaurant(ctx context.Context, id primitive.ObjectID, req *model.UpdateRestaurantRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	existingRestaurant, err := rs.restaurantRepo.GetByID(ctx, id)
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

	if err := rs.restaurantRepo.Update(ctx, existingRestaurant); err != nil {
		return nil, fmt.Errorf("failed to update restaurant: %w", err)
	}

	rs.logger.Info("Restaurant updated successfully", "restaurant_id", existingRestaurant.ID.Hex())
	return existingRestaurant, nil
}

// DeleteRestaurant deletes a restaurant.
func (rs *RestaurantService) DeleteRestaurant(ctx context.Context, id primitive.ObjectID) error {
	// Check if restaurant exists
	existingRestaurant, err := rs.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return fmt.Errorf("restaurant not found")
	}

	if err := rs.restaurantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete restaurant: %w", err)
	}

	rs.logger.Info("Restaurant deleted successfully", "restaurant_id", id.Hex())
	return nil
}

// ListRestaurants retrieves restaurants with filtering and pagination.
func (rs *RestaurantService) ListRestaurants(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*model.Restaurant, error) {
	restaurants, err := rs.restaurantRepo.List(ctx, filters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list restaurants: %w", err)
	}
	return restaurants, nil
}

// GetRestaurantsByIDs retrieves restaurants by a list of IDs.
func (rs *RestaurantService) GetRestaurantsByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*model.Restaurant, error) {
	if len(ids) == 0 {
		return []*model.Restaurant{}, nil
	}

	restaurants, err := rs.restaurantRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurants by IDs: %w", err)
	}
	return restaurants, nil
}

// GetRestaurantsByLocation retrieves restaurants by location.
func (rs *RestaurantService) GetRestaurantsByLocation(ctx context.Context, latitude, longitude, radius float64) ([]*model.Restaurant, error) {
	restaurants, err := rs.restaurantRepo.GetByLocation(ctx, latitude, longitude, radius)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurants by location: %w", err)
	}
	return restaurants, nil
}

// GetRestaurantRoom retrieves a specific room from a restaurant.
func (rs *RestaurantService) GetRestaurantRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID) (*model.Room, error) {
	rest, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
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
func (rs *RestaurantService) GetRestaurantRooms(ctx context.Context, restaurantID primitive.ObjectID) ([]model.Room, error) {
	rest, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	return rest.Rooms, nil
}

// GetActiveRestaurantRooms retrieves all active rooms for a restaurant.
func (rs *RestaurantService) GetActiveRestaurantRooms(ctx context.Context, restaurantID primitive.ObjectID) ([]model.Room, error) {
	rest, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	return rest.GetActiveRooms(), nil
}

// GetRestaurantByURLNameOrID retrieves a restaurant by URL name or ObjectID.
func (rs *RestaurantService) GetRestaurantByURLNameOrID(ctx context.Context, urlNameOrID string) (*model.Restaurant, error) {
	// Try to parse as ObjectID first
	if id, err := primitive.ObjectIDFromHex(urlNameOrID); err == nil {
		return rs.GetRestaurantByID(ctx, id)
	}

	// Otherwise treat as URL name
	rest, err := rs.restaurantRepo.GetByURLName(ctx, urlNameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant by URL name: %w", err)
	}
	if rest == nil {
		return nil, fmt.Errorf("restaurant not found")
	}
	return rest, nil
}

// CheckURLNameAvailability checks if a URL name is available.
func (rs *RestaurantService) CheckURLNameAvailability(ctx context.Context, urlName string) (*model.UrlNameAvailabilityResponse, error) {
	rest, err := rs.restaurantRepo.GetByURLName(ctx, urlName)
	if err != nil {
		return nil, fmt.Errorf("failed to check URL name availability: %w", err)
	}

	return &model.UrlNameAvailabilityResponse{
		IsAvailable: rest == nil,
	}, nil
}

// UpdateRestaurantSettings updates only the settings of a restaurant.
func (rs *RestaurantService) UpdateRestaurantSettings(ctx context.Context, id primitive.ObjectID, req *model.UpdateRestaurantSettingsRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	existingRestaurant, err := rs.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Update settings fields
	if req.AutoReservationAccepting != nil {
		existingRestaurant.Settings.AutoReservationAccepting = *req.AutoReservationAccepting
	}
	if req.MaxGuestsPerReservation != nil {
		existingRestaurant.Settings.MaxGuestsPerReservation = *req.MaxGuestsPerReservation
	}
	if req.DefaultReservationDuration != nil {
		existingRestaurant.Settings.DefaultReservationDuration = *req.DefaultReservationDuration
	}

	existingRestaurant.UpdatedAt = time.Now()

	if err := rs.restaurantRepo.Update(ctx, existingRestaurant); err != nil {
		return nil, fmt.Errorf("failed to update restaurant settings: %w", err)
	}

	rs.logger.Info("Restaurant settings updated successfully", "restaurant_id", existingRestaurant.ID.Hex())
	return existingRestaurant, nil
}

// EnableRestaurant enables a restaurant.
func (rs *RestaurantService) EnableRestaurant(ctx context.Context, id primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	existingRestaurant, err := rs.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	existingRestaurant.IsActive = true
	existingRestaurant.UpdatedAt = time.Now()

	if err := rs.restaurantRepo.Update(ctx, existingRestaurant); err != nil {
		return nil, fmt.Errorf("failed to enable restaurant: %w", err)
	}

	rs.logger.Info("Restaurant enabled successfully", "restaurant_id", existingRestaurant.ID.Hex())
	return existingRestaurant, nil
}

// DisableRestaurant disables a restaurant.
func (rs *RestaurantService) DisableRestaurant(ctx context.Context, id primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	existingRestaurant, err := rs.restaurantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if existingRestaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	existingRestaurant.IsActive = false
	existingRestaurant.UpdatedAt = time.Now()

	if err := rs.restaurantRepo.Update(ctx, existingRestaurant); err != nil {
		return nil, fmt.Errorf("failed to disable restaurant: %w", err)
	}

	rs.logger.Info("Restaurant disabled successfully", "restaurant_id", existingRestaurant.ID.Hex())
	return existingRestaurant, nil
}

// GetRestaurantAvailability gets availability information for a restaurant on a specific date.
func (rs *RestaurantService) GetRestaurantAvailability(ctx context.Context, urlNameOrID string, date string) (*model.RestaurantAvailabilityResponse, error) {
	// Get restaurant
	restaurant, err := rs.GetRestaurantByURLNameOrID(ctx, urlNameOrID)
	if err != nil {
		return nil, err
	}

	// If no date provided, use today
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Parse the date to get the day of week
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	// Check availability for each room
	roomAvailabilities := make([]model.RoomAvailability, 0, len(restaurant.Rooms))
	isRestaurantAvailable := false

	for _, room := range restaurant.Rooms {
		roomAvail := model.RoomAvailability{
			RoomID:   room.ID,
			RoomName: room.Name,
		}

		// Check if room is enabled
		if !room.IsEnabled {
			roomAvail.IsAvailable = false
			roomAvail.Reason = "room_disabled"
		} else if room.IsClosedOnDate(date) {
			// Check if room is closed on this specific date
			roomAvail.IsAvailable = false
			roomAvail.Reason = "closed_on_date"
		} else if !room.IsOpenOnDay(parsedDate.Weekday()) {
			// Check if room is open on this day of week
			roomAvail.IsAvailable = false
			roomAvail.Reason = "closed_on_weekday"
		} else if len(room.GetActiveTables()) == 0 {
			// Check if room has any active tables
			roomAvail.IsAvailable = false
			roomAvail.Reason = "no_active_tables"
		} else {
			// Room is available
			roomAvail.IsAvailable = true
			isRestaurantAvailable = true
		}

		roomAvailabilities = append(roomAvailabilities, roomAvail)
	}

	return &model.RestaurantAvailabilityResponse{
		RestaurantID: restaurant.ID,
		URLName:      restaurant.URLName,
		Date:         date,
		IsAvailable:  isRestaurantAvailable,
		Rooms:        roomAvailabilities,
	}, nil
}

// AddSubURL adds a new sub-URL to a restaurant.
func (rs *RestaurantService) AddSubURL(ctx context.Context, restaurantID primitive.ObjectID, req *model.CreateSubURLRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Check for duplicate name
	if restaurant.HasSubURLWithName(req.Name) {
		return nil, fmt.Errorf("sub-URL name already exists")
	}

	// Check for duplicate key
	if restaurant.HasSubURLWithKey(req.Key) {
		return nil, fmt.Errorf("sub-URL key already exists")
	}

	// Create new sub-URL with generated ID
	subURL := model.SubURL{
		ID:   primitive.NewObjectID(),
		Name: req.Name,
		Key:  req.Key,
	}

	// Add to restaurant's sub-URLs
	restaurant.SubURLs = append(restaurant.SubURLs, subURL)
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to add sub-URL: %w", err)
	}

	rs.logger.Info("Sub-URL added successfully", "restaurant_id", restaurant.ID.Hex(), "sub_url_id", subURL.ID)
	return restaurant, nil
}

// DeleteSubURL removes a sub-URL from a restaurant.
func (rs *RestaurantService) DeleteSubURL(ctx context.Context, restaurantID primitive.ObjectID, subURLID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Check if sub-URL exists
	subURL := restaurant.GetSubURL(subURLID)
	if subURL == nil {
		return nil, fmt.Errorf("sub-URL not found")
	}

	// Remove sub-URL from slice
	var updatedSubURLs []model.SubURL
	for _, sub := range restaurant.SubURLs {
		if sub.ID != subURLID {
			updatedSubURLs = append(updatedSubURLs, sub)
		}
	}
	restaurant.SubURLs = updatedSubURLs
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to delete sub-URL: %w", err)
	}

	rs.logger.Info("Sub-URL deleted successfully", "restaurant_id", restaurant.ID.Hex(), "sub_url_id", subURLID)
	return restaurant, nil
}

// AddRoom adds a new room to a restaurant.
func (rs *RestaurantService) AddRoom(ctx context.Context, restaurantID primitive.ObjectID, req *model.CreateRoomRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Check for duplicate room name
	for _, room := range restaurant.Rooms {
		if room.Name == req.Name {
			return nil, fmt.Errorf("room name already exists")
		}
	}

	// Set default enabled status if not provided
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	// Initialize tables slice if nil
	tables := req.Tables
	if tables == nil {
		tables = []model.Table{}
	}

	// Initialize closed dates slice if nil
	closedDates := req.ClosedDates
	if closedDates == nil {
		closedDates = []string{}
	}

	// Create new room with generated ID
	room := model.Room{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		IsEnabled:   isEnabled,
		Tables:      tables,
		WorkHours:   req.WorkHours,
		ClosedDates: closedDates,
	}

	// Add to restaurant's rooms
	restaurant.Rooms = append(restaurant.Rooms, room)
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to add room: %w", err)
	}

	rs.logger.Info("Room added successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", room.ID)
	return restaurant, nil
}

// UpdateRoom updates a room in a restaurant.
func (rs *RestaurantService) UpdateRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, req *model.UpdateRoomRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to update
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Check for duplicate room name (if changing name)
	if req.Name != nil {
		for i, room := range restaurant.Rooms {
			if i != roomIndex && room.Name == *req.Name {
				return nil, fmt.Errorf("room name already exists")
			}
		}
	}

	// Update room fields
	room := &restaurant.Rooms[roomIndex]
	if req.Name != nil {
		room.Name = *req.Name
	}
	if req.IsEnabled != nil {
		room.IsEnabled = *req.IsEnabled
	}
	if req.Tables != nil {
		room.Tables = req.Tables
	}
	if req.WorkHours != nil {
		room.WorkHours = *req.WorkHours
	}
	if req.ClosedDates != nil {
		room.ClosedDates = req.ClosedDates
	}

	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to update room: %w", err)
	}

	rs.logger.Info("Room updated successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID)
	return restaurant, nil
}

// DeleteRoom removes a room from a restaurant.
func (rs *RestaurantService) DeleteRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Check if room exists
	room := restaurant.GetRoom(roomID)
	if room == nil {
		return nil, fmt.Errorf("room not found")
	}

	// Remove room from slice
	var updatedRooms []model.Room
	for _, r := range restaurant.Rooms {
		if r.ID != roomID {
			updatedRooms = append(updatedRooms, r)
		}
	}
	restaurant.Rooms = updatedRooms
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to delete room: %w", err)
	}

	rs.logger.Info("Room deleted successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID)
	return restaurant, nil
}

// EnableRoom enables a room in a restaurant.
func (rs *RestaurantService) EnableRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to enable
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Enable the room
	restaurant.Rooms[roomIndex].IsEnabled = true
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to enable room: %w", err)
	}

	rs.logger.Info("Room enabled successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID)
	return restaurant, nil
}

// DisableRoom disables a room in a restaurant.
func (rs *RestaurantService) DisableRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to disable
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Disable the room
	restaurant.Rooms[roomIndex].IsEnabled = false
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to disable room: %w", err)
	}

	rs.logger.Info("Room disabled successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID)
	return restaurant, nil
}

// SaveElements saves multiple elements to a room.
func (rs *RestaurantService) SaveElements(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, req *model.SaveElementsRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to update
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Update elements - replace all elements with the new ones
	restaurant.Rooms[roomIndex].Elements = req.Elements
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to save elements: %w", err)
	}

	rs.logger.Info("Elements saved successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID, "element_count", len(req.Elements))
	return restaurant, nil
}

// UpdateElement updates a specific element in a room.
func (rs *RestaurantService) UpdateElement(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, elementID primitive.ObjectID, req *model.UpdateElementRequest) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to update
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Find the element to update
	elementIndex := -1
	for i, element := range restaurant.Rooms[roomIndex].Elements {
		if element.ID == elementID {
			elementIndex = i
			break
		}
	}
	if elementIndex == -1 {
		return nil, fmt.Errorf("element not found")
	}

	// Update element fields
	element := &restaurant.Rooms[roomIndex].Elements[elementIndex]
	if req.Type != nil {
		element.Type = *req.Type
	}
	if req.Code != nil {
		element.Code = *req.Code
	}
	if req.Index != nil {
		element.Index = *req.Index
	}
	if req.Rotation != nil {
		element.Rotation = *req.Rotation
	}
	if req.TableIndex != nil {
		element.TableIndex = req.TableIndex
	}
	if req.X != nil {
		element.X = *req.X
	}
	if req.Y != nil {
		element.Y = *req.Y
	}
	if req.Width != nil {
		element.Width = *req.Width
	}
	if req.Height != nil {
		element.Height = *req.Height
	}
	if req.Text != nil {
		element.Text = req.Text
	}
	if req.FontSize != nil {
		element.FontSize = req.FontSize
	}
	if req.IsEnabled != nil {
		element.IsEnabled = *req.IsEnabled
	}
	if req.Seats != nil {
		element.Seats = req.Seats
	}

	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to update element: %w", err)
	}

	rs.logger.Info("Element updated successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID, "element_id", elementID)
	return restaurant, nil
}

// EnableElement enables a specific element in a room.
func (rs *RestaurantService) EnableElement(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, elementID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to update
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Find the element to enable
	elementIndex := -1
	for i, element := range restaurant.Rooms[roomIndex].Elements {
		if element.ID == elementID {
			elementIndex = i
			break
		}
	}
	if elementIndex == -1 {
		return nil, fmt.Errorf("element not found")
	}

	// Enable the element
	restaurant.Rooms[roomIndex].Elements[elementIndex].IsEnabled = true
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to enable element: %w", err)
	}

	rs.logger.Info("Element enabled successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID, "element_id", elementID)
	return restaurant, nil
}

// DisableElement disables a specific element in a room.
func (rs *RestaurantService) DisableElement(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, elementID primitive.ObjectID) (*model.Restaurant, error) {
	// Get existing restaurant
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Find the room to update
	roomIndex := -1
	for i, room := range restaurant.Rooms {
		if room.ID == roomID {
			roomIndex = i
			break
		}
	}
	if roomIndex == -1 {
		return nil, fmt.Errorf("room not found")
	}

	// Find the element to disable
	elementIndex := -1
	for i, element := range restaurant.Rooms[roomIndex].Elements {
		if element.ID == elementID {
			elementIndex = i
			break
		}
	}
	if elementIndex == -1 {
		return nil, fmt.Errorf("element not found")
	}

	// Disable the element
	restaurant.Rooms[roomIndex].Elements[elementIndex].IsEnabled = false
	restaurant.UpdatedAt = time.Now()

	// Update restaurant in database
	if err := rs.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to disable element: %w", err)
	}

	rs.logger.Info("Element disabled successfully", "restaurant_id", restaurant.ID.Hex(), "room_id", roomID, "element_id", elementID)
	return restaurant, nil
}

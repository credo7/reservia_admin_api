// Package restaurant contains the restaurant domain entities and logic.
package restaurant

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Restaurant represents a restaurant in the system.
type Restaurant struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	URLName     string             `json:"url_name" bson:"url_name"`
	Address     string             `json:"address" bson:"address"`
	Phone       string             `json:"phone" bson:"phone"`
	Email       string             `json:"email" bson:"email"`
	Description string             `json:"description" bson:"description"`
	IsActive    bool               `json:"is_active" bson:"is_active"`
	UTCOffset   int                `json:"utc_offset" bson:"utc_offset"`
	Settings    Settings           `json:"settings" bson:"settings"`
	Rooms       []Room             `json:"rooms" bson:"rooms"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Settings contains restaurant configuration.
type Settings struct {
	AutoReservationAccepting   bool `json:"auto_reservation_accepting" bson:"auto_reservation_accepting"`
	MaxGuestsPerReservation    int  `json:"max_guests_per_reservation" bson:"max_guests_per_reservation"`
	DefaultReservationDuration int  `json:"default_reservation_duration" bson:"default_reservation_duration"` // minutes
}

// Room represents a room within a restaurant.
type Room struct {
	ID          string    `json:"id" bson:"id"`
	Name        string    `json:"name" bson:"name"`
	IsEnabled   bool      `json:"is_enabled" bson:"is_enabled"`
	Tables      []Table   `json:"tables" bson:"tables"`
	WorkHours   WorkHours `json:"work_hours" bson:"work_hours"`
	ClosedDates []string  `json:"closed_dates" bson:"closed_dates"` // YYYY-MM-DD format
}

// Table represents a table within a room.
type Table struct {
	ID        string `json:"id" bson:"id"`
	Name      string `json:"name" bson:"name"`
	Capacity  int    `json:"capacity" bson:"capacity"`
	IsEnabled bool   `json:"is_enabled" bson:"is_enabled"`
}

// WorkHours represents working hours for a room.
type WorkHours struct {
	Monday    DayHours `json:"monday" bson:"monday"`
	Tuesday   DayHours `json:"tuesday" bson:"tuesday"`
	Wednesday DayHours `json:"wednesday" bson:"wednesday"`
	Thursday  DayHours `json:"thursday" bson:"thursday"`
	Friday    DayHours `json:"friday" bson:"friday"`
	Saturday  DayHours `json:"saturday" bson:"saturday"`
	Sunday    DayHours `json:"sunday" bson:"sunday"`
}

// DayHours represents opening hours for a specific day.
type DayHours struct {
	IsOpen    bool   `json:"is_open" bson:"is_open"`
	OpenTime  string `json:"open_time" bson:"open_time"`   // HH:MM format
	CloseTime string `json:"close_time" bson:"close_time"` // HH:MM format
}

// CreateRestaurantRequest represents a request to create a new restaurant.
type CreateRestaurantRequest struct {
	Name        string   `json:"name" validate:"required,min=2,max=100"`
	URLName     string   `json:"url_name" validate:"required,min=2,max=50,alphanum"`
	Address     string   `json:"address" validate:"required,min=5,max=200"`
	Phone       string   `json:"phone" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Description string   `json:"description" validate:"max=500"`
	UTCOffset   int      `json:"utc_offset" validate:"min=-12,max=14"`
	Settings    Settings `json:"settings"`
	Rooms       []Room   `json:"rooms"`
}

// UpdateRestaurantRequest represents a request to update a restaurant.
type UpdateRestaurantRequest struct {
	Name        *string   `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Address     *string   `json:"address,omitempty" validate:"omitempty,min=5,max=200"`
	Phone       *string   `json:"phone,omitempty"`
	Email       *string   `json:"email,omitempty" validate:"omitempty,email"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
	IsActive    *bool     `json:"is_active,omitempty"`
	UTCOffset   *int      `json:"utc_offset,omitempty" validate:"omitempty,min=-12,max=14"`
	Settings    *Settings `json:"settings,omitempty"`
	Rooms       []Room    `json:"rooms,omitempty"`
}

// RestaurantResponse represents the response when returning restaurant data.
type RestaurantResponse struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	URLName     string             `json:"url_name"`
	Address     string             `json:"address"`
	Phone       string             `json:"phone"`
	Email       string             `json:"email"`
	Description string             `json:"description"`
	IsActive    bool               `json:"is_active"`
	UTCOffset   int                `json:"utc_offset"`
	Settings    Settings           `json:"settings"`
	RoomCount   int                `json:"room_count"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// ToResponse converts a Restaurant to RestaurantResponse.
func (r *Restaurant) ToResponse() *RestaurantResponse {
	return &RestaurantResponse{
		ID:          r.ID,
		Name:        r.Name,
		URLName:     r.URLName,
		Address:     r.Address,
		Phone:       r.Phone,
		Email:       r.Email,
		Description: r.Description,
		IsActive:    r.IsActive,
		UTCOffset:   r.UTCOffset,
		Settings:    r.Settings,
		RoomCount:   len(r.Rooms),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// GetRoom returns a room by ID.
func (r *Restaurant) GetRoom(roomID string) *Room {
	for i := range r.Rooms {
		if r.Rooms[i].ID == roomID {
			return &r.Rooms[i]
		}
	}
	return nil
}

// GetActiveRooms returns all active rooms.
func (r *Restaurant) GetActiveRooms() []Room {
	var activeRooms []Room
	for _, room := range r.Rooms {
		if room.IsEnabled {
			activeRooms = append(activeRooms, room)
		}
	}
	return activeRooms
}

// GetTable returns a table by ID within a specific room.
func (room *Room) GetTable(tableID string) *Table {
	for i := range room.Tables {
		if room.Tables[i].ID == tableID {
			return &room.Tables[i]
		}
	}
	return nil
}

// GetActiveTables returns all active tables in the room.
func (room *Room) GetActiveTables() []Table {
	var activeTables []Table
	for _, table := range room.Tables {
		if table.IsEnabled {
			activeTables = append(activeTables, table)
		}
	}
	return activeTables
}

// GetTotalCapacity returns the total capacity of all active tables in the room.
func (room *Room) GetTotalCapacity() int {
	total := 0
	for _, table := range room.Tables {
		if table.IsEnabled {
			total += table.Capacity
		}
	}
	return total
}

// IsOpenOnDay checks if the room is open on a specific day.
func (room *Room) IsOpenOnDay(day time.Weekday) bool {
	var dayHours DayHours
	switch day {
	case time.Monday:
		dayHours = room.WorkHours.Monday
	case time.Tuesday:
		dayHours = room.WorkHours.Tuesday
	case time.Wednesday:
		dayHours = room.WorkHours.Wednesday
	case time.Thursday:
		dayHours = room.WorkHours.Thursday
	case time.Friday:
		dayHours = room.WorkHours.Friday
	case time.Saturday:
		dayHours = room.WorkHours.Saturday
	case time.Sunday:
		dayHours = room.WorkHours.Sunday
	}
	return dayHours.IsOpen
}

// IsClosedOnDate checks if the room is closed on a specific date.
func (room *Room) IsClosedOnDate(date string) bool {
	for _, closedDate := range room.ClosedDates {
		if closedDate == date {
			return true
		}
	}
	return false
}

// Package restaurant contains the restaurant model entities and logic.
package model

import (
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Restaurant represents a restaurant in the system.
type Restaurant struct {
	ID                           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	Name                         string              `json:"name" bson:"name"`
	URLName                      string              `json:"urlName" bson:"url_name"`
	City                         string              `json:"city" bson:"city"`
	Address                      string              `json:"address" bson:"address"`
	Phone                        string              `json:"phone" bson:"phone"`
	Email                        string              `json:"email" bson:"email"`
	Description                  string              `json:"description" bson:"description"`
	IsActive                     bool                `json:"isActive" bson:"is_active"`
	UTCOffset                    int                 `json:"utcOffset" bson:"utc_offset"`
	IsDefaultScheduleForAllRooms bool                `json:"isDefaultScheduleForAllRooms" bson:"is_default_schedule_for_all_rooms"`
	IsSpecialScheduleForAllRooms bool                `json:"isSpecialScheduleForAllRooms" bson:"is_special_schedule_for_all_rooms"`
	IsClosedForAllRooms          bool                `json:"isClosedForAllRooms" bson:"is_closed_for_all_rooms"`
	DefaultRoomID                *primitive.ObjectID `json:"defaultRoomId,omitempty" bson:"default_room_id,omitempty"`
	Settings                     Settings            `json:"settings" bson:"settings"`
	Rooms                        []Room              `json:"rooms" bson:"rooms"`
	SubURLs                      []SubURL            `json:"subUrls" bson:"sub_urls"`
	CreatedAt                    time.Time           `json:"createdAt" bson:"created_at"`
	UpdatedAt                    time.Time           `json:"updatedAt" bson:"updated_at"`
}

// Settings contains restaurant configuration (matches Python RestaurantSettingsSchema).
type Settings struct {
	AutoReservationAccepting   bool   `json:"autoReservationAccepting" bson:"auto_reservation_accepting"`
	ReservationIntervalMinutes int    `json:"reservationIntervalMinutes" bson:"reservation_interval_minutes"`
	ReservationReminderMinutes int    `json:"reservationReminderMinutes" bson:"reservation_reminder_minutes"`
	MinReservationDuration     string `json:"minReservationDuration" bson:"min_reservation_duration"` // HH:MM format
	MaxReservationDuration     string `json:"maxReservationDuration" bson:"max_reservation_duration"` // HH:MM format
	MaxDaysInAdvance           int    `json:"maxDaysInAdvance" bson:"max_days_in_advance"`
	// Legacy fields for backward compatibility (these will be deprecated)
	MaxGuestsPerReservation    int `json:"maxGuestsPerReservation,omitempty" bson:"max_guests_per_reservation,omitempty"`
	DefaultReservationDuration int `json:"defaultReservationDuration,omitempty" bson:"default_reservation_duration,omitempty"` // minutes
}

// Room represents a room within a restaurant.
type Room struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	IsEnabled   bool               `json:"isEnabled" bson:"is_enabled"`
	Tables      []Table            `json:"tables" bson:"tables"`
	Elements    []Element          `json:"elements" bson:"elements"`
	WorkHours   WorkHours          `json:"workHours" bson:"work_hours"`
	ClosedDates []string           `json:"closedDates" bson:"closed_dates"` // YYYY-MM-DD format
}

// Table represents a table within a room.
type Table struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Capacity  int                `json:"capacity" bson:"capacity"`
	IsEnabled bool               `json:"isEnabled" bson:"is_enabled"`
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
	IsOpen    bool   `json:"isOpen" bson:"is_open"`
	OpenTime  string `json:"openTime" bson:"open_time"`   // HH:MM format
	CloseTime string `json:"closeTime" bson:"close_time"` // HH:MM format
}

// CreateRestaurantRequest represents a request to create a new restaurant.
// This matches the Python CreateRestaurantBodySchema with minimal required fields.
type CreateRestaurantRequest struct {
	Name    string `json:"name" validate:"required,min=2,max=100"`
	Phone   string `json:"phone" validate:"required"`
	Address string `json:"address" validate:"required,min=5,max=200"`
	City    string `json:"city" validate:"required,min=1,max=100"`
}

// UpdateRestaurantRequest represents a request to update a restaurant.
type UpdateRestaurantRequest struct {
	Name        *string   `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Address     *string   `json:"address,omitempty" validate:"omitempty,min=5,max=200"`
	Phone       *string   `json:"phone,omitempty"`
	Email       *string   `json:"email,omitempty" validate:"omitempty,email"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
	IsActive    *bool     `json:"isActive,omitempty"`
	UTCOffset   *int      `json:"utcOffset,omitempty" validate:"omitempty,min=-12,max=14"`
	Settings    *Settings `json:"settings,omitempty"`
	Rooms       []Room    `json:"rooms,omitempty"`
}

// RestaurantResponse represents the response when returning restaurant data.
type RestaurantResponse struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	URLName     string             `json:"urlName"`
	Address     string             `json:"address"`
	Phone       string             `json:"phone"`
	Email       string             `json:"email"`
	Description string             `json:"description"`
	IsActive    bool               `json:"isActive"`
	UTCOffset   int                `json:"utcOffset"`
	Settings    Settings           `json:"settings"`
	RoomCount   int                `json:"roomCount"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
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
func (r *Restaurant) GetRoom(roomID primitive.ObjectID) *Room {
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
func (room *Room) GetTable(tableID primitive.ObjectID) *Table {
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

// Additional models for new endpoints

// UrlNameAvailabilityRequest represents a request to check URL name availability.
type UrlNameAvailabilityRequest struct {
	URLName string `json:"urlName" validate:"required,min=2,max=50,alphanum"`
}

// UrlNameAvailabilityResponse represents the availability check response.
type UrlNameAvailabilityResponse struct {
	IsAvailable bool `json:"isAvailable"`
}

// UpdateRestaurantSettingsRequest represents a request to update restaurant settings only.
type UpdateRestaurantSettingsRequest struct {
	AutoReservationAccepting   *bool `json:"autoReservationAccepting,omitempty"`
	MaxGuestsPerReservation    *int  `json:"maxGuestsPerReservation,omitempty"`
	DefaultReservationDuration *int  `json:"defaultReservationDuration,omitempty"`
}

// RestaurantMiniResponse represents a minimal restaurant response for lists.
type RestaurantMiniResponse struct {
	ID       primitive.ObjectID `json:"id"`
	Name     string             `json:"name"`
	URLName  string             `json:"urlName"`
	IsActive bool               `json:"isActive"`
}

// ToMiniResponse converts a Restaurant to RestaurantMiniResponse.
func (r *Restaurant) ToMiniResponse() *RestaurantMiniResponse {
	return &RestaurantMiniResponse{
		ID:       r.ID,
		Name:     r.Name,
		URLName:  r.URLName,
		IsActive: r.IsActive,
	}
}

// RestaurantAvailabilityResponse represents restaurant availability for a specific date.
type RestaurantAvailabilityResponse struct {
	RestaurantID primitive.ObjectID `json:"restaurantId"`
	URLName      string             `json:"urlName"`
	Date         string             `json:"date"` // YYYY-MM-DD format
	IsAvailable  bool               `json:"isAvailable"`
	Rooms        []RoomAvailability `json:"rooms"`
}

// RoomAvailability represents availability for a specific room.
type RoomAvailability struct {
	RoomID      primitive.ObjectID `json:"roomId"`
	RoomName    string             `json:"roomName"`
	IsAvailable bool               `json:"isAvailable"`
	Reason      string             `json:"reason,omitempty"` // closed, no_tables, etc.
}

// SubURL represents a sub-URL for tracking reservation sources.
type SubURL struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
	Key  string             `json:"key" bson:"key"`
}

// CreateSubURLRequest represents a request to create a new sub-URL.
type CreateSubURLRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
	Key  string `json:"key" validate:"required,min=1,max=50"`
}

// CreateRoomRequest represents a request to create a new room.
type CreateRoomRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=100"`
	IsEnabled   *bool     `json:"isEnabled,omitempty"`
	Tables      []Table   `json:"tables,omitempty"`
	WorkHours   WorkHours `json:"workHours"`
	ClosedDates []string  `json:"closedDates,omitempty"`
}

// UpdateRoomRequest represents a request to update a room.
type UpdateRoomRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	IsEnabled   *bool      `json:"isEnabled,omitempty"`
	Tables      []Table    `json:"tables,omitempty"`
	WorkHours   *WorkHours `json:"workHours,omitempty"`
	ClosedDates []string   `json:"closedDates,omitempty"`
}

// GetSubURL returns a sub-URL by ID.
func (r *Restaurant) GetSubURL(subURLID primitive.ObjectID) *SubURL {
	for i := range r.SubURLs {
		if r.SubURLs[i].ID == subURLID {
			return &r.SubURLs[i]
		}
	}
	return nil
}

// HasSubURLWithName checks if a sub-URL with the given name already exists.
func (r *Restaurant) HasSubURLWithName(name string) bool {
	for _, subURL := range r.SubURLs {
		if subURL.Name == name {
			return true
		}
	}
	return false
}

// HasSubURLWithKey checks if a sub-URL with the given key already exists.
func (r *Restaurant) HasSubURLWithKey(key string) bool {
	for _, subURL := range r.SubURLs {
		if subURL.Key == key {
			return true
		}
	}
	return false
}

// Element types and codes
type ElementType string

const (
	ElementTypeTable ElementType = "TABLE"
	ElementTypeSeat  ElementType = "SEAT"
	ElementTypeWall  ElementType = "WALL"
	ElementTypeText  ElementType = "TEXT"
	ElementTypeArrow ElementType = "ARROW"
)

type ElementCode string

const (
	ElementCodeWindow                ElementCode = "WINDOW"
	ElementCodeWall                  ElementCode = "WALL"
	ElementCodeRoundedWall           ElementCode = "ROUNDED_WALL"
	ElementCodePolygonalWall         ElementCode = "POLYGONAL_WALL"
	ElementCodeChair                 ElementCode = "CHAIR"
	ElementCodeArmchair1             ElementCode = "ARMCHAIR_1"
	ElementCodeArmchair2             ElementCode = "ARMCHAIR_2"
	ElementCodeSquareTable           ElementCode = "SQUARE_TABLE"
	ElementCodeRoundedTable          ElementCode = "ROUNDED_TABLE"
	ElementCodeText                  ElementCode = "TEXT"
	ElementCodeArrow                 ElementCode = "ARROW"
	ElementCodeChairSlot             ElementCode = "CHAIR_SLOT"
	ElementCodeActiveChairSlot       ElementCode = "ACTIVE_CHAIR_SLOT"
	ElementCodeSource                ElementCode = "SOURCE"
	ElementCodeSofaLeftArmLong       ElementCode = "SOFA_LEFT_ARM_LONG"
	ElementCodeSofaLeftArmShort      ElementCode = "SOFA_LEFT_ARM_SHORT"
	ElementCodeSofaLeftNoArm         ElementCode = "SOFA_LEFT_NO_ARM"
	ElementCodeSofaSemiLeftArmLong   ElementCode = "SOFA_SEMI_LEFT_ARM_LONG"
	ElementCodeSofaSemiLeftArmShort  ElementCode = "SOFA_SEMI_LEFT_ARM_SHORT"
	ElementCodeSofaSemiLeftNoArm     ElementCode = "SOFA_SEMI_LEFT_NO_ARM"
	ElementCodeSofaRightArmLong      ElementCode = "SOFA_RIGHT_ARM_LONG"
	ElementCodeSofaRightArmShort     ElementCode = "SOFA_RIGHT_ARM_SHORT"
	ElementCodeSofaRightNoArm        ElementCode = "SOFA_RIGHT_NO_ARM"
	ElementCodeSofaSemiRightArmLong  ElementCode = "SOFA_SEMI_RIGHT_ARM_LONG"
	ElementCodeSofaSemiRightArmShort ElementCode = "SOFA_SEMI_RIGHT_ARM_SHORT"
	ElementCodeSofaSemiRightNoArm    ElementCode = "SOFA_SEMI_RIGHT_NO_ARM"
	ElementCodeSofaRightConnector    ElementCode = "SOFA_RIGHT_CONNECTOR"
	ElementCodeSofaMiddleConnector   ElementCode = "SOFA_MIDDLE_CONNECTOR"
	ElementCodeSofaLeftConnector     ElementCode = "SOFA_LEFT_CONNECTOR"
)

// Element represents a table, seat, or other UI element within a room.
type Element struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RoomID     primitive.ObjectID `json:"roomId" bson:"room_id"`
	Type       ElementType        `json:"type" bson:"type"`
	Code       ElementCode        `json:"code" bson:"code"`
	Index      int                `json:"index" bson:"index"`
	Rotation   int                `json:"rotation" bson:"rotation"`
	TableIndex *int               `json:"tableIndex,omitempty" bson:"table_index,omitempty"`
	X          float64            `json:"x" bson:"x"`
	Y          float64            `json:"y" bson:"y"`
	Width      int                `json:"width" bson:"width"`
	Height     int                `json:"height" bson:"height"`
	Text       *string            `json:"text,omitempty" bson:"text,omitempty"`
	FontSize   *int               `json:"fontSize,omitempty" bson:"font_size,omitempty"`
	IsEnabled  bool               `json:"isEnabled" bson:"is_enabled"`
	Seats      []Element          `json:"seats,omitempty" bson:"seats,omitempty"`
}

// CreateElementRequest represents a request to create a new element.
type CreateElementRequest struct {
	Type       ElementType `json:"type" validate:"required"`
	Code       ElementCode `json:"code" validate:"required"`
	Index      int         `json:"index" validate:"required"`
	Rotation   int         `json:"rotation"`
	TableIndex *int        `json:"tableIndex,omitempty"`
	X          float64     `json:"x" validate:"required"`
	Y          float64     `json:"y" validate:"required"`
	Width      int         `json:"width" validate:"required,gt=0"`
	Height     int         `json:"height" validate:"required,gt=0"`
	Text       *string     `json:"text,omitempty"`
	FontSize   *int        `json:"fontSize,omitempty"`
	Seats      []Element   `json:"seats,omitempty"`
}

// UpdateElementRequest represents a request to update an element.
type UpdateElementRequest struct {
	Type       *ElementType `json:"type,omitempty"`
	Code       *ElementCode `json:"code,omitempty"`
	Index      *int         `json:"index,omitempty"`
	Rotation   *int         `json:"rotation,omitempty"`
	TableIndex *int         `json:"tableIndex,omitempty"`
	X          *float64     `json:"x,omitempty"`
	Y          *float64     `json:"y,omitempty"`
	Width      *int         `json:"width,omitempty"`
	Height     *int         `json:"height,omitempty"`
	Text       *string      `json:"text,omitempty"`
	FontSize   *int         `json:"fontSize,omitempty"`
	IsEnabled  *bool        `json:"isEnabled,omitempty"`
	Seats      []Element    `json:"seats,omitempty"`
}

// SaveElementsRequest represents a request to save multiple elements.
type SaveElementsRequest struct {
	Elements []Element `json:"elements" validate:"required"`
}

// GetElement returns an element by ID within a specific room.
func (room *Room) GetElement(elementID primitive.ObjectID) *Element {
	for i := range room.Elements {
		if room.Elements[i].ID == elementID {
			return &room.Elements[i]
		}
	}
	return nil
}

// GetActiveElements returns all active elements in the room.
func (room *Room) GetActiveElements() []Element {
	var activeElements []Element
	for _, element := range room.Elements {
		if element.IsEnabled {
			activeElements = append(activeElements, element)
		}
	}
	return activeElements
}

// GetTablesFromElements returns all table elements in the room.
func (room *Room) GetTablesFromElements() []Element {
	var tables []Element
	for _, element := range room.Elements {
		if element.Type == ElementTypeTable {
			tables = append(tables, element)
		}
	}
	return tables
}

// GenerateURLName generates a URL-friendly name from the restaurant name.
// This follows Python's approach of creating a slug from the restaurant name.
func GenerateURLName(name string) string {
	// Convert to lowercase
	urlName := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	urlName = reg.ReplaceAllString(urlName, "-")

	// Remove leading and trailing hyphens
	urlName = strings.Trim(urlName, "-")

	// Limit length to 50 characters
	if len(urlName) > 50 {
		urlName = urlName[:50]
		// Remove trailing hyphen if we cut in the middle of a word
		urlName = strings.TrimRight(urlName, "-")
	}

	return urlName
}

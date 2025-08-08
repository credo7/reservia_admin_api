// Package restaurant contains the restaurant model entities and logic.
package model

import (
	"fmt"
	"regexp"
	"strconv"
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
	ID                   primitive.ObjectID    `json:"id" bson:"_id,omitempty"`
	Name                 string                `json:"name" bson:"name"`
	IsEnabled            bool                  `json:"isEnabled" bson:"is_enabled"`
	Tables               []Table               `json:"tables" bson:"tables"`
	Elements             []Element             `json:"elements" bson:"elements"`
	RegularSchedule      WeekSchedule          `json:"regularSchedule" bson:"regular_schedule"`
	SpecialDateSchedules []SpecialDateSchedule `json:"specialDateSchedules" bson:"special_date_schedules"`
	ClosedDates          []string              `json:"closedDates" bson:"closed_dates"` // YYYY-MM-DD format
}

// Table represents a table within a room.
type Table struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Capacity  int                `json:"capacity" bson:"capacity"`
	IsEnabled bool               `json:"isEnabled" bson:"is_enabled"`
}


// Enhanced Schedule Models (matches Python implementation)

// WeekSchedule represents a weekly schedule with support for multiple time ranges per day.
type WeekSchedule struct {
	Monday    WeekDaySchedule `json:"monday" bson:"monday"`
	Tuesday   WeekDaySchedule `json:"tuesday" bson:"tuesday"`
	Wednesday WeekDaySchedule `json:"wednesday" bson:"wednesday"`
	Thursday  WeekDaySchedule `json:"thursday" bson:"thursday"`
	Friday    WeekDaySchedule `json:"friday" bson:"friday"`
	Saturday  WeekDaySchedule `json:"saturday" bson:"saturday"`
	Sunday    WeekDaySchedule `json:"sunday" bson:"sunday"`
}

// WeekDaySchedule represents a single day's schedule with multiple time ranges.
type WeekDaySchedule struct {
	IsActive   bool        `json:"isActive" bson:"is_active"`
	TimeRanges []TimeRange `json:"timeRanges" bson:"time_ranges"`
}

// TimeRange represents a single time period within a day.
type TimeRange struct {
	StartTime string `json:"startTime" bson:"start_time"` // HH:MM format
	EndTime   string `json:"endTime" bson:"end_time"`     // HH:MM format
}

// SpecialDateSchedule represents override schedules for specific dates.
type SpecialDateSchedule struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name       string             `json:"name" bson:"name"`                   // e.g., "New Year Special Hours"
	Dates      []string           `json:"dates" bson:"dates"`                 // YYYY-MM-DD format
	TimeRanges []TimeRange        `json:"timeRanges" bson:"time_ranges"`
	IsActive   bool               `json:"isActive" bson:"is_active"`
}

// CreateRestaurantRequest represents a request to create a new restaurant.
// This matches the Python CreateRestaurantBodySchema with minimal required fields.
type CreateRestaurantRequest struct {
	Name    string             `json:"name" validate:"required,min=2,max=100"`
	Phone   string             `json:"phone" validate:"required"`
	Address string             `json:"address" validate:"required,min=5,max=200"`
	CityID  primitive.ObjectID `json:"cityId" validate:"required" bson:"city_id"`
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

// RestaurantSummary represents a summary response when returning restaurant data (matches Python RestaurantMiniSchema).
type RestaurantSummary struct {
	ID        primitive.ObjectID `json:"id"`
	Name      string             `json:"name"`
	URLName   string             `json:"urlName"`
	Address   string             `json:"address"`
	City      string             `json:"city"`
	Phone     string             `json:"phone"`
	UTCOffset int                `json:"utcOffset"`
	IsActive  bool               `json:"isActive"`
	// Role would be added when we know the employee's role for this restaurant
}

// ToSummary converts a Restaurant to RestaurantSummary (matches Python RestaurantMiniSchema).
func (r *Restaurant) ToSummary() *RestaurantSummary {
	return &RestaurantSummary{
		ID:        r.ID,
		Name:      r.Name,
		URLName:   r.URLName,
		Address:   r.Address,
		City:      r.City,
		Phone:     r.Phone,
		UTCOffset: r.UTCOffset,
		IsActive:  r.IsActive,
		// Role would be set by the caller based on employee's role for this restaurant
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


// IsClosedOnDate checks if the room is closed on a specific date.
func (room *Room) IsClosedOnDate(date string) bool {
	for _, closedDate := range room.ClosedDates {
		if closedDate == date {
			return true
		}
	}
	return false
}

// Enhanced Schedule Methods (matching Python implementation)

// IsOpenOnDate checks if the room is open on a specific date using enhanced schedule.
func (room *Room) IsOpenOnDate(checkDate time.Time) bool {
	if !room.IsEnabled {
		return false
	}

	dateStr := checkDate.Format("2006-01-02")

	// Check if room is closed on this date
	if room.IsClosedOnDate(dateStr) {
		return false
	}

	// Check for overnight shifts from previous day
	prevDate := checkDate.AddDate(0, 0, -1)
	if room.HasOvernightShiftIntoDate(prevDate, checkDate) {
		return true
	}

	// Check regular schedule for this date
	return room.HasWorkingHoursOnDate(checkDate)
}

// GetActiveTimeRangesForDate returns active time ranges for a specific date with priority system.
func (room *Room) GetActiveTimeRangesForDate(checkDate time.Time) []TimeRange {
	dateStr := checkDate.Format("2006-01-02")

	// 1. First check special schedules (highest priority)
	for _, special := range room.SpecialDateSchedules {
		if !special.IsActive {
			continue
		}
		
		for _, specialDate := range special.Dates {
			if specialDate == dateStr {
				return special.TimeRanges
			}
		}
	}

	// 2. Fall back to regular weekly schedule
	daySchedule := room.GetDayScheduleForDate(checkDate)
	if daySchedule.IsActive {
		return daySchedule.TimeRanges
	}

	return []TimeRange{}
}

// GetDayScheduleForDate gets the regular schedule for a specific weekday.
func (room *Room) GetDayScheduleForDate(checkDate time.Time) WeekDaySchedule {
	switch checkDate.Weekday() {
	case time.Monday:
		return room.RegularSchedule.Monday
	case time.Tuesday:
		return room.RegularSchedule.Tuesday
	case time.Wednesday:
		return room.RegularSchedule.Wednesday
	case time.Thursday:
		return room.RegularSchedule.Thursday
	case time.Friday:
		return room.RegularSchedule.Friday
	case time.Saturday:
		return room.RegularSchedule.Saturday
	case time.Sunday:
		return room.RegularSchedule.Sunday
	default:
		return WeekDaySchedule{}
	}
}

// HasWorkingHoursOnDate checks if room has working hours on a specific date.
func (room *Room) HasWorkingHoursOnDate(checkDate time.Time) bool {
	timeRanges := room.GetActiveTimeRangesForDate(checkDate)
	return len(timeRanges) > 0
}

// HasOvernightShiftIntoDate checks if there's an overnight shift from previous day extending into current day.
func (room *Room) HasOvernightShiftIntoDate(dayBefore, currentDay time.Time) bool {
	timeRanges := room.GetActiveTimeRangesForDate(dayBefore)

	for _, tr := range timeRanges {
		startTime, err := time.Parse("15:04", tr.StartTime)
		if err != nil {
			continue
		}
		
		endTime, err := time.Parse("15:04", tr.EndTime)
		if err != nil {
			continue
		}

		// If end <= start, it's an overnight shift
		if endTime.Hour() < startTime.Hour() || (endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
			// This is an overnight shift
			endDateTime := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), endTime.Hour(), endTime.Minute(), 0, 0, currentDay.Location())

			// Check if it extends into current day
			currentDayStart := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), 0, 0, 0, 0, currentDay.Location())
			if endDateTime.After(currentDayStart) {
				return true
			}
		}
	}

	return false
}

// IsWithinWorkingHours checks if a specific time falls within room's working hours.
func (room *Room) IsWithinWorkingHours(checkDateTime time.Time) bool {
	if !room.IsEnabled {
		return false
	}

	dateStr := checkDateTime.Format("2006-01-02")
	timeStr := checkDateTime.Format("15:04")

	// Check if room is closed on this date
	if room.IsClosedOnDate(dateStr) {
		return false
	}

	// Get active time ranges for this date
	timeRanges := room.GetActiveTimeRangesForDate(checkDateTime)

	for _, tr := range timeRanges {
		startTime, err := time.Parse("15:04", tr.StartTime)
		if err != nil {
			continue
		}
		
		endTime, err := time.Parse("15:04", tr.EndTime)
		if err != nil {
			continue
		}

		checkTime, err := time.Parse("15:04", timeStr)
		if err != nil {
			continue
		}

		// Handle normal time range (within same day)
		if endTime.After(startTime) {
			if (checkTime.Equal(startTime) || checkTime.After(startTime)) && checkTime.Before(endTime) {
				return true
			}
		} else {
			// Handle overnight shift (spans midnight)
			if checkTime.Equal(startTime) || checkTime.After(startTime) {
				return true
			}
			if checkTime.Before(endTime) {
				return true
			}
		}
	}

	// Check if current time is within overnight shift from previous day
	prevDay := checkDateTime.AddDate(0, 0, -1)
	prevTimeRanges := room.GetActiveTimeRangesForDate(prevDay)

	for _, tr := range prevTimeRanges {
		startTime, err := time.Parse("15:04", tr.StartTime)
		if err != nil {
			continue
		}
		
		endTime, err := time.Parse("15:04", tr.EndTime)
		if err != nil {
			continue
		}

		checkTime, err := time.Parse("15:04", timeStr)
		if err != nil {
			continue
		}

		// Check if this is an overnight shift that extends into current day
		if endTime.Hour() < startTime.Hour() || (endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
			if checkTime.Before(endTime) {
				return true
			}
		}
	}

	return false
}

// GetNextWorkingPeriod returns the next working period for the room starting from the given time.
func (room *Room) GetNextWorkingPeriod(fromDateTime time.Time) (time.Time, time.Time, bool) {
	if !room.IsEnabled {
		return time.Time{}, time.Time{}, false
	}

	// Check current day and next 7 days
	for i := 0; i < 7; i++ {
		checkDate := fromDateTime.AddDate(0, 0, i)
		
		if room.IsClosedOnDate(checkDate.Format("2006-01-02")) {
			continue
		}

		timeRanges := room.GetActiveTimeRangesForDate(checkDate)
		
		for _, tr := range timeRanges {
			startTime, err := time.Parse("15:04", tr.StartTime)
			if err != nil {
				continue
			}
			
			endTime, err := time.Parse("15:04", tr.EndTime)
			if err != nil {
				continue
			}

			startDateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, checkDate.Location())
			var endDateTime time.Time

			// Handle overnight shifts
			if endTime.Hour() < startTime.Hour() || (endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
				endDateTime = time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day()+1, endTime.Hour(), endTime.Minute(), 0, 0, checkDate.Location())
			} else {
				endDateTime = time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, checkDate.Location())
			}

			// If this period is in the future, return it
			if startDateTime.After(fromDateTime) || startDateTime.Equal(fromDateTime) {
				return startDateTime, endDateTime, true
			}
		}
	}

	return time.Time{}, time.Time{}, false
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
	Name                 string                `json:"name" validate:"required,min=1,max=100"`
	IsEnabled            *bool                 `json:"isEnabled,omitempty"`
	Tables               []Table               `json:"tables,omitempty"`
	RegularSchedule      *WeekSchedule         `json:"regularSchedule,omitempty"`
	SpecialDateSchedules []SpecialDateSchedule `json:"specialDateSchedules,omitempty"`
	ClosedDates          []string              `json:"closedDates,omitempty"`
}

// UpdateRoomRequest represents a request to update a room.
type UpdateRoomRequest struct {
	Name                 *string               `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	IsEnabled            *bool                 `json:"isEnabled,omitempty"`
	Tables               []Table               `json:"tables,omitempty"`
	RegularSchedule      *WeekSchedule         `json:"regularSchedule,omitempty"`
	SpecialDateSchedules []SpecialDateSchedule `json:"specialDateSchedules,omitempty"`
	ClosedDates          []string              `json:"closedDates,omitempty"`
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

// Enhanced Availability Models (matching Python RestaurantAvailability)

// DayAvailability represents availability for a specific day (matches Python DayAvailability).
type DayAvailability struct {
	Date     string `json:"date"`     // YYYY-MM-DD format
	Weekday  string `json:"weekday"`  // "Monday", "Tuesday", etc.
	IsActive bool   `json:"isActive"` // Whether bookings are available on this day
}

// ReservationSlot represents a bookable time slot (matches Python ReservationSlotSchema).
type ReservationSlot struct {
	RoomID  primitive.ObjectID `json:"roomId" bson:"room_id"`
	TableID primitive.ObjectID `json:"tableId" bson:"table_id"`
	StartAt time.Time          `json:"startAt" bson:"start_at"`
	EndAt   time.Time          `json:"endAt" bson:"end_at"`
}

// RestaurantAvailability represents complete restaurant availability (matches Python RestaurantAvailability).
type RestaurantAvailability struct {
	ID                        primitive.ObjectID             `json:"id"`
	Name                      string                         `json:"name"`
	City                      string                         `json:"city"`
	Address                   string                         `json:"address"`
	ShowDates                 []DayAvailability              `json:"showDates"`                 // Available booking dates
	SelectedDate              string                         `json:"selectedDate"`              // YYYY-MM-DD format  
	SlotsByTableID            map[string][]ReservationSlot   `json:"slotsByTableId"`            // Available slots per table
	StartTimeToMaxDurationMap map[string][]string            `json:"startTimeToMaxDurationMap"` // Time slots → available durations
}

// TableAvailabilityFilters represents filters for checking table availability.
type TableAvailabilityFilters struct {
	RestaurantID primitive.ObjectID `json:"restaurantId"`
	RoomID       primitive.ObjectID `json:"roomId,omitempty"`
	TableID      primitive.ObjectID `json:"tableId,omitempty"`
	StartAt      time.Time          `json:"startAt"`
	EndAt        time.Time          `json:"endAt"`
	Date         time.Time          `json:"date"`
}

// GetActiveStatuses returns the reservation statuses that block table availability.
// Matches Python active statuses: PENDING, CONFIRMED, ARRIVED
func GetActiveReservationStatuses() []Status {
	return []Status{StatusPending, StatusConfirmed, StatusArrived}
}

// ParseDuration parses duration string like "2:30" into time.Duration.
func ParseDuration(duration string) (time.Duration, error) {
	parts := strings.Split(duration, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}
	
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hours in duration: %s", parts[0])
	}
	
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in duration: %s", parts[1])
	}
	
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}

// FormatDuration formats time.Duration into "H:MM" format.
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

// GetMinReservationDuration returns the minimum reservation duration as time.Duration.
func (s *Settings) GetMinReservationDuration() (time.Duration, error) {
	return ParseDuration(s.MinReservationDuration)
}

// GetMaxReservationDuration returns the maximum reservation duration as time.Duration.
func (s *Settings) GetMaxReservationDuration() (time.Duration, error) {
	return ParseDuration(s.MaxReservationDuration)
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

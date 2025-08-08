// Package service provides business logic for room availability calculations.
package service

import (
	"context"
	"time"

	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RoomAvailabilityService handles room availability calculations matching Python implementation.
type RoomAvailabilityService struct {
	logger logger.Logger
}

// NewRoomAvailabilityService creates a new room availability service.
func NewRoomAvailabilityService(logger logger.Logger) *RoomAvailabilityService {
	return &RoomAvailabilityService{
		logger: logger,
	}
}

// CalculateRoomAvailabilityForDate calculates room availability for a specific date.
// This matches the Python _calculate_availability_slots functionality.
func (s *RoomAvailabilityService) CalculateRoomAvailabilityForDate(
	ctx context.Context,
	restaurant *model.Restaurant,
	room *model.Room,
	checkDate time.Time,
) model.RoomAvailability {
	roomAvailability := model.RoomAvailability{
		RoomID:   room.ID,
		RoomName: room.Name,
	}

	// Check if room is enabled
	if !room.IsEnabled {
		roomAvailability.IsAvailable = false
		roomAvailability.Reason = "disabled"
		return roomAvailability
	}

	// Check if room is open on this date
	if !room.IsOpenOnDate(checkDate) {
		roomAvailability.IsAvailable = false
		roomAvailability.Reason = "closed"
		return roomAvailability
	}

	// Check if room has active tables
	activeTables := room.GetActiveTables()
	if len(activeTables) == 0 {
		roomAvailability.IsAvailable = false
		roomAvailability.Reason = "no_tables"
		return roomAvailability
	}

	// Room is available
	roomAvailability.IsAvailable = true
	return roomAvailability
}

// CalculateRestaurantAvailabilityForDate calculates availability for all rooms in a restaurant.
// This matches the Python restaurant availability calculation functionality.
func (s *RoomAvailabilityService) CalculateRestaurantAvailabilityForDate(
	ctx context.Context,
	restaurant *model.Restaurant,
	checkDate time.Time,
) model.RestaurantAvailabilityBasicResponse {
	dateStr := checkDate.Format("2006-01-02")
	
	response := model.RestaurantAvailabilityBasicResponse{
		RestaurantID: restaurant.ID,
		URLName:      restaurant.URLName,
		Date:         dateStr,
		IsAvailable:  false,
		Rooms:        make([]model.RoomAvailability, 0, len(restaurant.Rooms)),
	}

	// Check if restaurant is active
	if !restaurant.IsActive {
		s.logger.Debug("Restaurant is not active", "restaurant_id", restaurant.ID)
		return response
	}

	var hasAvailableRoom bool

	// Calculate availability for each room
	for _, room := range restaurant.Rooms {
		roomAvailability := s.CalculateRoomAvailabilityForDate(ctx, restaurant, &room, checkDate)
		response.Rooms = append(response.Rooms, roomAvailability)

		if roomAvailability.IsAvailable {
			hasAvailableRoom = true
		}
	}

	response.IsAvailable = hasAvailableRoom

	s.logger.Debug("Calculated restaurant availability",
		"restaurant_id", restaurant.ID,
		"date", dateStr,
		"is_available", response.IsAvailable,
		"available_rooms", s.countAvailableRooms(response.Rooms),
	)

	return response
}

// GetRoomWorkingHoursForDate returns the working periods for a room on a specific date.
// This matches the Python _get_active_time_ranges_for_date functionality.
func (s *RoomAvailabilityService) GetRoomWorkingHoursForDate(
	ctx context.Context,
	room *model.Room,
	checkDate time.Time,
) []WorkingPeriod {
	if !room.IsEnabled {
		return []WorkingPeriod{}
	}

	dateStr := checkDate.Format("2006-01-02")

	// Check if room is closed on this date
	if room.IsClosedOnDate(dateStr) {
		return []WorkingPeriod{}
	}

	timeRanges := room.GetActiveTimeRangesForDate(checkDate)
	var workingPeriods []WorkingPeriod

	for _, tr := range timeRanges {
		startTime, err := time.Parse("15:04", tr.StartTime)
		if err != nil {
			s.logger.Error("Failed to parse start time", "error", err, "time", tr.StartTime)
			continue
		}

		endTime, err := time.Parse("15:04", tr.EndTime)
		if err != nil {
			s.logger.Error("Failed to parse end time", "error", err, "time", tr.EndTime)
			continue
		}

		// Create datetime objects for the working period
		startDateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(),
			startTime.Hour(), startTime.Minute(), 0, 0, checkDate.Location())

		var endDateTime time.Time
		// Handle overnight shifts (when end time is before start time)
		if endTime.Hour() < startTime.Hour() || (endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
			// This is an overnight shift - end time is on the next day
			endDateTime = time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day()+1,
				endTime.Hour(), endTime.Minute(), 0, 0, checkDate.Location())
		} else {
			// Normal working hours within the same day
			endDateTime = time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(),
				endTime.Hour(), endTime.Minute(), 0, 0, checkDate.Location())
		}

		workingPeriods = append(workingPeriods, WorkingPeriod{
			StartTime: startDateTime,
			EndTime:   endDateTime,
			IsOvernight: endTime.Hour() < startTime.Hour() || 
				(endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()),
		})
	}

	// Also check for overnight shifts from the previous day that extend into current day
	prevDate := checkDate.AddDate(0, 0, -1)
	if room.HasOvernightShiftIntoDate(prevDate, checkDate) {
		prevTimeRanges := room.GetActiveTimeRangesForDate(prevDate)
		for _, tr := range prevTimeRanges {
			startTime, err := time.Parse("15:04", tr.StartTime)
			if err != nil {
				continue
			}

			endTime, err := time.Parse("15:04", tr.EndTime)
			if err != nil {
				continue
			}

			// Check if this is an overnight shift
			if endTime.Hour() < startTime.Hour() || (endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
				startDateTime := time.Date(prevDate.Year(), prevDate.Month(), prevDate.Day(),
					startTime.Hour(), startTime.Minute(), 0, 0, prevDate.Location())
				endDateTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(),
					endTime.Hour(), endTime.Minute(), 0, 0, checkDate.Location())

				// Only add if it actually extends into the current day
				currentDayStart := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), 0, 0, 0, 0, checkDate.Location())
				if endDateTime.After(currentDayStart) {
					workingPeriods = append(workingPeriods, WorkingPeriod{
						StartTime:   startDateTime,
						EndTime:     endDateTime,
						IsOvernight: true,
						FromPreviousDay: true,
					})
				}
			}
		}
	}

	return workingPeriods
}

// IsRoomAvailableAtTime checks if a room is available at a specific time.
// This matches the Python room availability checking functionality.
func (s *RoomAvailabilityService) IsRoomAvailableAtTime(
	ctx context.Context,
	room *model.Room,
	checkDateTime time.Time,
) bool {
	if !room.IsEnabled {
		return false
	}

	// Use the enhanced IsWithinWorkingHours method
	return room.IsWithinWorkingHours(checkDateTime)
}

// GetNextAvailableTime finds the next time when a room becomes available.
// This matches the Python _get_next_working_period functionality.
func (s *RoomAvailabilityService) GetNextAvailableTime(
	ctx context.Context,
	room *model.Room,
	fromDateTime time.Time,
) (time.Time, bool) {
	startTime, _, found := room.GetNextWorkingPeriod(fromDateTime)
	return startTime, found
}

// ValidateReservationTime checks if a reservation time is valid for a room.
// This includes checking working hours, closed dates, and overnight shifts.
func (s *RoomAvailabilityService) ValidateReservationTime(
	ctx context.Context,
	room *model.Room,
	reservationStart time.Time,
	reservationEnd time.Time,
) (bool, string) {
	if !room.IsEnabled {
		return false, "room is disabled"
	}

	// Check if reservation start time is within working hours
	if !s.IsRoomAvailableAtTime(ctx, room, reservationStart) {
		return false, "reservation start time is outside working hours"
	}

	// For reservations that span multiple days, we need to check each day
	currentCheck := reservationStart
	for currentCheck.Before(reservationEnd) {
		// Check if the current time is within working hours
		if !room.IsWithinWorkingHours(currentCheck) {
			return false, "reservation spans non-working hours"
		}

		// Move to next hour for checking (could be optimized)
		currentCheck = currentCheck.Add(time.Hour)

		// If we've checked past the end, break
		if currentCheck.After(reservationEnd) {
			break
		}
	}

	// Check the end time as well
	if !room.IsWithinWorkingHours(reservationEnd) {
		return false, "reservation end time is outside working hours"
	}

	return true, ""
}

// WorkingPeriod represents a working period for a room.
type WorkingPeriod struct {
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
	IsOvernight     bool      `json:"isOvernight"`
	FromPreviousDay bool      `json:"fromPreviousDay,omitempty"`
}

// countAvailableRooms counts the number of available rooms.
func (s *RoomAvailabilityService) countAvailableRooms(rooms []model.RoomAvailability) int {
	count := 0
	for _, room := range rooms {
		if room.IsAvailable {
			count++
		}
	}
	return count
}

// ApplyRestaurantWideSchedule applies restaurant-wide schedule changes to all rooms.
// This matches the Python restaurant-wide schedule enforcement functionality.
func (s *RoomAvailabilityService) ApplyRestaurantWideSchedule(
	ctx context.Context,
	restaurant *model.Restaurant,
	updateType string,
	scheduleData interface{},
) error {
	switch updateType {
	case "regular_schedule":
		if restaurant.IsDefaultScheduleForAllRooms {
			if regularSchedule, ok := scheduleData.(model.WeekSchedule); ok {
				for i := range restaurant.Rooms {
					restaurant.Rooms[i].RegularSchedule = regularSchedule
				}
				s.logger.Info("Applied regular schedule to all rooms", "restaurant_id", restaurant.ID)
			}
		}

	case "special_schedule":
		if restaurant.IsSpecialScheduleForAllRooms {
			if specialSchedules, ok := scheduleData.([]model.SpecialDateSchedule); ok {
				for i := range restaurant.Rooms {
					restaurant.Rooms[i].SpecialDateSchedules = specialSchedules
				}
				s.logger.Info("Applied special schedule to all rooms", "restaurant_id", restaurant.ID)
			}
		}

	case "closed_dates":
		if restaurant.IsClosedForAllRooms {
			if closedDates, ok := scheduleData.([]string); ok {
				for i := range restaurant.Rooms {
					restaurant.Rooms[i].ClosedDates = closedDates
				}
				s.logger.Info("Applied closed dates to all rooms", "restaurant_id", restaurant.ID)
			}
		}
	}

	return nil
}

// GetRoomScheduleSummary provides a summary of room schedule configuration.
func (s *RoomAvailabilityService) GetRoomScheduleSummary(
	ctx context.Context,
	room *model.Room,
) RoomScheduleSummary {
	summary := RoomScheduleSummary{
		RoomID:   room.ID,
		RoomName: room.Name,
		IsEnabled: room.IsEnabled,
		HasRegularSchedule: s.hasActiveRegularSchedule(room),
		SpecialScheduleCount: len(room.SpecialDateSchedules),
		ClosedDatesCount: len(room.ClosedDates),
	}

	// Count active special schedules
	activeSpecialCount := 0
	for _, special := range room.SpecialDateSchedules {
		if special.IsActive {
			activeSpecialCount++
		}
	}
	summary.ActiveSpecialScheduleCount = activeSpecialCount

	return summary
}

// RoomScheduleSummary provides summary information about a room's schedule configuration.
type RoomScheduleSummary struct {
	RoomID                    primitive.ObjectID `json:"roomId"`
	RoomName                  string             `json:"roomName"`
	IsEnabled                 bool               `json:"isEnabled"`
	HasRegularSchedule        bool               `json:"hasRegularSchedule"`
	SpecialScheduleCount      int                `json:"specialScheduleCount"`
	ActiveSpecialScheduleCount int               `json:"activeSpecialScheduleCount"`
	ClosedDatesCount          int                `json:"closedDatesCount"`
}

// hasActiveRegularSchedule checks if the room has any active regular schedule.
func (s *RoomAvailabilityService) hasActiveRegularSchedule(room *model.Room) bool {
	return room.RegularSchedule.Monday.IsActive ||
		room.RegularSchedule.Tuesday.IsActive ||
		room.RegularSchedule.Wednesday.IsActive ||
		room.RegularSchedule.Thursday.IsActive ||
		room.RegularSchedule.Friday.IsActive ||
		room.RegularSchedule.Saturday.IsActive ||
		room.RegularSchedule.Sunday.IsActive
}
// Package service provides enhanced restaurant availability functionality matching Python implementation.
package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RestaurantAvailabilityService handles comprehensive restaurant availability calculations
// matching the Python RestaurantAvailabilityService functionality.
type RestaurantAvailabilityService struct {
	restaurantRepo  repository.RestaurantRepository
	reservationRepo repository.ReservationRepository
	logger          logger.Logger
}

// NewRestaurantAvailabilityService creates a new restaurant availability service.
func NewRestaurantAvailabilityService(
	restaurantRepo repository.RestaurantRepository,
	reservationRepo repository.ReservationRepository,
	logger logger.Logger,
) *RestaurantAvailabilityService {
	return &RestaurantAvailabilityService{
		restaurantRepo:  restaurantRepo,
		reservationRepo: reservationRepo,
		logger:          logger,
	}
}

// GetAvailability returns complete restaurant availability for a specific date (matches Python get_availability).
func (s *RestaurantAvailabilityService) GetAvailability(
	ctx context.Context,
	urlNameOrID string,
	chosenDate time.Time,
) (*model.RestaurantAvailability, error) {
	// Get restaurant by URL name or ID
	restaurant, err := s.getRestaurantByIdentifier(ctx, urlNameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}

	// Calculate current time with restaurant's UTC offset
	currentTime := s.getCurrentTimeWithOffset(restaurant)

	response := &model.RestaurantAvailability{
		ID:                        restaurant.ID,
		Name:                      restaurant.Name,
		City:                      restaurant.City,
		Address:                   restaurant.Address,
		SelectedDate:              chosenDate.Format("2006-01-02"),
		SlotsByTableID:            make(map[string][]model.ReservationSlot),
		StartTimeToMaxDurationMap: make(map[string][]string),
	}

	// Generate show dates (available booking dates)
	response.ShowDates = s.getShowDates(restaurant, currentTime)

	// If restaurant is not active, return empty availability
	if !restaurant.IsActive {
		s.logger.Debug("Restaurant is not active", "restaurant_id", restaurant.ID)
		return response, nil
	}

	// Calculate availability slots for the chosen date
	err = s.calculateAvailabilitySlots(ctx, restaurant, chosenDate, response)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate availability slots: %w", err)
	}

	// Filter slots for current day (remove past slots if it's today)
	if chosenDate.Format("2006-01-02") == currentTime.Format("2006-01-02") {
		s.filterSlotsForToday(response, currentTime)
	}

	// Build start time to duration mapping for UI
	s.buildTimeSlotMapping(response, restaurant.Settings)

	s.logger.Debug("Calculated restaurant availability",
		"restaurant_id", restaurant.ID,
		"date", chosenDate.Format("2006-01-02"),
		"total_slots", s.countTotalSlots(response.SlotsByTableID),
	)

	return response, nil
}

// calculateAvailabilitySlots calculates available slots for all rooms and tables (matches Python _calculate_availability_slots).
func (s *RestaurantAvailabilityService) calculateAvailabilitySlots(
	ctx context.Context,
	restaurant *model.Restaurant,
	chosenDate time.Time,
	response *model.RestaurantAvailability,
) error {
	for _, room := range restaurant.GetActiveRooms() {
		if !room.IsOpenOnDate(chosenDate) {
			s.logger.Debug("Room is not open on date", 
				"room_id", room.ID, 
				"date", chosenDate.Format("2006-01-02"))
			continue
		}

		// Get active time ranges for the chosen date
		timeRanges := room.GetActiveTimeRangesForDate(chosenDate)
		if len(timeRanges) == 0 {
			continue
		}

		// Process each time range
		for _, timeRange := range timeRanges {
			err := s.processRoomAvailabilityForTimeRange(ctx, restaurant, &room, chosenDate, timeRange, response)
			if err != nil {
				s.logger.Error("Failed to process room availability", 
					"error", err, 
					"room_id", room.ID,
					"time_range", fmt.Sprintf("%s-%s", timeRange.StartTime, timeRange.EndTime))
				continue
			}
		}
	}

	return nil
}

// processRoomAvailabilityForTimeRange processes availability for a room within a specific time range
// (matches Python _process_room_availability).
func (s *RestaurantAvailabilityService) processRoomAvailabilityForTimeRange(
	ctx context.Context,
	restaurant *model.Restaurant,
	room *model.Room,
	chosenDate time.Time,
	timeRange model.TimeRange,
	response *model.RestaurantAvailability,
) error {
	// Parse time range
	startAt, endAt, err := s.parseTimeRangeForDate(chosenDate, timeRange)
	if err != nil {
		return fmt.Errorf("failed to parse time range: %w", err)
	}

	// Get all active tables in the room
	activeTables := room.GetActiveTables()
	if len(activeTables) == 0 {
		return nil
	}

	// Collect table IDs
	var tableIDs []primitive.ObjectID
	for _, table := range activeTables {
		tableIDs = append(tableIDs, table.ID)
	}

	// Get existing reservations for all tables in this time period
	existingReservations, err := s.reservationRepo.GetActiveReservationsForMultipleTables(
		ctx, restaurant.ID, tableIDs, startAt, endAt,
	)
	if err != nil {
		return fmt.Errorf("failed to get existing reservations: %w", err)
	}

	// Group reservations by table ID
	reservationsByTable := s.groupReservationsByTable(existingReservations)

	// Calculate availability for each table
	for _, table := range activeTables {
		tableIDStr := table.ID.Hex()
		tableReservations := reservationsByTable[table.ID]

		// Calculate free slots for this table
		slots := s.calculateTableSlots(startAt, endAt, tableReservations, room.ID, table.ID, restaurant.Settings)
		
		if len(slots) > 0 {
			response.SlotsByTableID[tableIDStr] = append(response.SlotsByTableID[tableIDStr], slots...)
		}
	}

	return nil
}

// calculateTableSlots calculates available time slots for a table by finding gaps between reservations
// (matches Python gap-finding algorithm).
func (s *RestaurantAvailabilityService) calculateTableSlots(
	startAt, endAt time.Time,
	reservations []*model.Reservation,
	roomID, tableID primitive.ObjectID,
	settings model.Settings,
) []model.ReservationSlot {
	var slots []model.ReservationSlot

	// Get minimum reservation duration
	minDuration, err := settings.GetMinReservationDuration()
	if err != nil {
		s.logger.Error("Failed to parse min reservation duration", "error", err)
		minDuration = time.Hour // Default to 1 hour
	}

	// Sort reservations by start time
	sort.Slice(reservations, func(i, j int) bool {
		return reservations[i].StartAt.Before(reservations[j].StartAt)
	})

	currentTime := startAt

	// Find gaps between reservations
	for _, reservation := range reservations {
		// If there's a gap before this reservation
		if reservation.StartAt.After(currentTime) {
			// Calculate gap duration
			gapDuration := reservation.StartAt.Sub(currentTime)
			
			// Only add slot if gap is >= minimum duration
			if gapDuration >= minDuration {
				slots = append(slots, model.ReservationSlot{
					RoomID:  roomID,
					TableID: tableID,
					StartAt: currentTime,
					EndAt:   reservation.StartAt,
				})
			}
		}

		// Move current time to end of this reservation
		if reservation.EndAt.After(currentTime) {
			currentTime = reservation.EndAt
		}
	}

	// Check for gap after the last reservation
	if endAt.After(currentTime) {
		gapDuration := endAt.Sub(currentTime)
		if gapDuration >= minDuration {
			slots = append(slots, model.ReservationSlot{
				RoomID:  roomID,
				TableID: tableID,
				StartAt: currentTime,
				EndAt:   endAt,
			})
		}
	}

	return slots
}

// buildTimeSlotMapping builds the start time to duration mapping for UI
// (matches Python _build_time_slot_mapping).
func (s *RestaurantAvailabilityService) buildTimeSlotMapping(
	response *model.RestaurantAvailability,
	settings model.Settings,
) {
	intervalMinutes := settings.ReservationIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 30 // Default
	}

	minDuration, err := settings.GetMinReservationDuration()
	if err != nil {
		minDuration = time.Hour
	}

	maxDuration, err := settings.GetMaxReservationDuration()  
	if err != nil {
		maxDuration = 3 * time.Hour
	}

	timeSlotMap := make(map[string][]string)

	// Process all available slots
	for _, slots := range response.SlotsByTableID {
		for _, slot := range slots {
			// Round start time to next interval boundary
			startTime := s.roundTimeToInterval(slot.StartAt, intervalMinutes)
			
			// Generate start times within this slot
			for currentStart := startTime; currentStart.Before(slot.EndAt); currentStart = currentStart.Add(time.Duration(intervalMinutes) * time.Minute) {
				if currentStart.Before(slot.StartAt) {
					continue
				}

				startTimeStr := currentStart.Format("15:04")
				
				// Generate available durations for this start time
				var durations []string
				maxEndTime := slot.EndAt
				if currentStart.Add(maxDuration).Before(maxEndTime) {
					maxEndTime = currentStart.Add(maxDuration)
				}

				for duration := minDuration; currentStart.Add(duration).Before(maxEndTime) || currentStart.Add(duration).Equal(maxEndTime); duration += time.Duration(intervalMinutes) * time.Minute {
					durationStr := model.FormatDuration(duration)
					durations = append(durations, durationStr)
				}

				// Merge with existing durations for this start time
				if len(durations) > 0 {
					existing := timeSlotMap[startTimeStr]
					timeSlotMap[startTimeStr] = s.mergeDurations(existing, durations)
				}
			}
		}
	}

	response.StartTimeToMaxDurationMap = timeSlotMap
}

// filterSlotsForToday removes past time slots for the current day (matches Python _filter_slots_for_today).
func (s *RestaurantAvailabilityService) filterSlotsForToday(
	response *model.RestaurantAvailability,
	currentTime time.Time,
) {
	for tableID, slots := range response.SlotsByTableID {
		var filteredSlots []model.ReservationSlot
		
		for _, slot := range slots {
			// Only keep slots that end after current time
			if slot.EndAt.After(currentTime) {
				// If slot starts before current time, adjust start time
				if slot.StartAt.Before(currentTime) {
					slot.StartAt = currentTime
				}
				filteredSlots = append(filteredSlots, slot)
			}
		}
		
		response.SlotsByTableID[tableID] = filteredSlots
	}
}

// getShowDates generates available booking dates (matches Python _get_show_dates).
func (s *RestaurantAvailabilityService) getShowDates(
	restaurant *model.Restaurant,
	currentTime time.Time,
) []model.DayAvailability {
	var showDates []model.DayAvailability
	
	maxDays := restaurant.Settings.MaxDaysInAdvance
	if maxDays <= 0 {
		maxDays = 31 // Default
	}

	for i := 0; i < maxDays; i++ {
		date := currentTime.AddDate(0, 0, i)
		
		showDates = append(showDates, model.DayAvailability{
			Date:     date.Format("2006-01-02"),
			Weekday:  date.Weekday().String(),
			IsActive: s.isDateBookable(restaurant, date),
		})
	}

	return showDates
}

// Helper methods

// getRestaurantByIdentifier gets restaurant by URL name or ObjectID.
func (s *RestaurantAvailabilityService) getRestaurantByIdentifier(ctx context.Context, identifier string) (*model.Restaurant, error) {
	// Try as ObjectID first
	if objectID, err := primitive.ObjectIDFromHex(identifier); err == nil {
		return s.restaurantRepo.GetByID(ctx, objectID)
	}
	
	// Try as URL name
	return s.restaurantRepo.GetByURLName(ctx, identifier)
}

// getCurrentTimeWithOffset calculates current time with restaurant's UTC offset.
func (s *RestaurantAvailabilityService) getCurrentTimeWithOffset(restaurant *model.Restaurant) time.Time {
	utc := time.Now().UTC()
	return utc.Add(time.Duration(restaurant.UTCOffset) * time.Hour)
}

// parseTimeRangeForDate converts time range strings to datetime objects for a specific date.
func (s *RestaurantAvailabilityService) parseTimeRangeForDate(
	date time.Time,
	timeRange model.TimeRange,
) (time.Time, time.Time, error) {
	startTime, err := time.Parse("15:04", timeRange.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start time: %w", err)
	}

	endTime, err := time.Parse("15:04", timeRange.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end time: %w", err)
	}

	startAt := time.Date(date.Year(), date.Month(), date.Day(), 
		startTime.Hour(), startTime.Minute(), 0, 0, date.Location())
	
	endAt := time.Date(date.Year(), date.Month(), date.Day(),
		endTime.Hour(), endTime.Minute(), 0, 0, date.Location())

	// Handle overnight shifts
	if endTime.Hour() < startTime.Hour() || 
		(endTime.Hour() == startTime.Hour() && endTime.Minute() <= startTime.Minute()) {
		endAt = endAt.AddDate(0, 0, 1)
	}

	return startAt, endAt, nil
}

// groupReservationsByTable groups reservations by table ID.
func (s *RestaurantAvailabilityService) groupReservationsByTable(
	reservations []*model.Reservation,
) map[primitive.ObjectID][]*model.Reservation {
	groups := make(map[primitive.ObjectID][]*model.Reservation)
	
	for _, reservation := range reservations {
		groups[reservation.TableID] = append(groups[reservation.TableID], reservation)
	}
	
	return groups
}

// roundTimeToInterval rounds time to the next interval boundary.
func (s *RestaurantAvailabilityService) roundTimeToInterval(t time.Time, intervalMinutes int) time.Time {
	minutes := t.Minute()
	roundedMinutes := int(math.Ceil(float64(minutes)/float64(intervalMinutes))) * intervalMinutes
	
	if roundedMinutes >= 60 {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, roundedMinutes-60, 0, 0, t.Location())
	}
	
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), roundedMinutes, 0, 0, t.Location())
}

// mergeDurations merges two duration slices, keeping unique values.
func (s *RestaurantAvailabilityService) mergeDurations(existing, new []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	// Add existing durations
	for _, duration := range existing {
		if !seen[duration] {
			result = append(result, duration)
			seen[duration] = true
		}
	}
	
	// Add new durations
	for _, duration := range new {
		if !seen[duration] {
			result = append(result, duration)
			seen[duration] = true
		}
	}
	
	return result
}

// isDateBookable checks if a date is bookable for the restaurant.
func (s *RestaurantAvailabilityService) isDateBookable(restaurant *model.Restaurant, date time.Time) bool {
	// Check if any room is open on this date
	for _, room := range restaurant.GetActiveRooms() {
		if room.IsOpenOnDate(date) {
			return true
		}
	}
	return false
}

// countTotalSlots counts total slots across all tables.
func (s *RestaurantAvailabilityService) countTotalSlots(slotsByTable map[string][]model.ReservationSlot) int {
	total := 0
	for _, slots := range slotsByTable {
		total += len(slots)
	}
	return total
}
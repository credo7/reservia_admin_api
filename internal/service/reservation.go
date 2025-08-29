// Package service provides reservation-related services.
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"reservia-admin-api/internal/model"
	"reservia-admin-api/internal/repository"
	"reservia-admin-api/pkg/logger"
)

// ReservationService handles reservation-related business logic.
type ReservationService struct {
	reservationRepo repository.ReservationRepository
	restaurantRepo  repository.RestaurantRepository
	logger          logger.Logger
}

// NewReservationService creates a new reservation service.
func NewReservationService(reservationRepo repository.ReservationRepository, restaurantRepo repository.RestaurantRepository, logger logger.Logger) *ReservationService {
	return &ReservationService{
		reservationRepo: reservationRepo,
		restaurantRepo:  restaurantRepo,
		logger:          logger,
	}
}

// CreateReservationByEmployee creates a new reservation by employee (admin).
func (rs *ReservationService) CreateReservationByEmployee(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, req *model.CreateReservationRequest, employeeID primitive.ObjectID) (*model.Reservation, error) {
	// Validate restaurant and room exist
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	room := restaurant.GetRoom(roomID)
	if room == nil {
		return nil, fmt.Errorf("room not found")
	}

	// Validate table exists and is active
	table := room.GetTable(req.TableID)
	if table == nil {
		return nil, fmt.Errorf("table not found")
	}
	if !table.IsEnabled {
		return nil, fmt.Errorf("table is not active")
	}

	// Parse duration and calculate end time
	endAt, err := rs.parseDurationAndCalculateEndTime(req.StartAt, req.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration format: %w", err)
	}

	// Generate unique reservation code
	code := rs.generateReservationCode()

	// Create reservation
	reservation := &model.Reservation{
		Code:               code,
		RestaurantID:       restaurantID,
		RoomID:             roomID,
		TableID:            req.TableID,
		FullName:           req.FullName,
		Email:              req.Email,
		Phone:              req.Phone,
		GuestCount:         req.GuestCount,
		UserNotes:          req.UserNotes,
		Status:             model.StatusConfirmed, // Admin reservations are automatically confirmed
		AuthorizeMethod:    model.AuthorizeMethodAdmin,
		IsUsedForAuth:      false,
		StartAt:            req.StartAt,
		EndAt:              endAt,
		InitialEndAt:       endAt,
		WorkingDayDate:     time.Date(req.StartAt.Year(), req.StartAt.Month(), req.StartAt.Day(), 0, 0, 0, 0, req.StartAt.Location()),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		AuthorizedAt:       &time.Time{},
		ConfirmedAt:        &time.Time{},
		AdminNotes:         fmt.Sprintf("Created by employee %s", employeeID.Hex()),
		IsSeen:             true, // Admin-created reservations are marked as seen
		TgUsername:         "",
		ExternalBookingID:  "",
		CancellationReason: "",
	}

	now := time.Now()
	reservation.AuthorizedAt = &now
	reservation.ConfirmedAt = &now

	// Check for conflicts
	conflicts, err := rs.checkForConflicts(ctx, reservation)
	if err != nil {
		return nil, fmt.Errorf("failed to check for conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("reservation conflicts with existing reservations at this time slot")
	}

	// Create reservation in database
	if err := rs.reservationRepo.Create(ctx, reservation); err != nil {
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	rs.logger.Info("Reservation created by employee", "reservation_id", reservation.ID.Hex(), "employee_id", employeeID.Hex(), "restaurant_id", restaurantID.Hex())
	return reservation, nil
}

// GetReservationsByRoom retrieves reservations for a specific room with optional filtering.
func (rs *ReservationService) GetReservationsByRoom(ctx context.Context, restaurantID primitive.ObjectID, roomID primitive.ObjectID, filters map[string]interface{}, limit, offset int) ([]*model.Reservation, error) {
	// Validate restaurant and room exist
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	room := restaurant.GetRoom(roomID)
	if room == nil {
		return nil, fmt.Errorf("room not found")
	}

	// For now, we'll get all reservations for the restaurant and filter by room ID
	// In a production system, we'd want to add room-specific methods to the repository
	allReservations, err := rs.reservationRepo.GetByRestaurantID(ctx, restaurantID, 1000, 0) // Get more to filter
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations: %w", err)
	}

	// Filter by room ID
	var roomReservations []*model.Reservation
	for _, reservation := range allReservations {
		if reservation.RoomID == roomID {
			roomReservations = append(roomReservations, reservation)
		}
	}

	// Apply additional filters if provided
	if len(filters) > 0 {
		roomReservations = rs.applyFilters(roomReservations, filters)
	}

	// Apply pagination
	start := offset
	end := offset + limit

	if start > len(roomReservations) {
		return []*model.Reservation{}, nil
	}

	if end > len(roomReservations) {
		end = len(roomReservations)
	}

	return roomReservations[start:end], nil
}

// Helper methods

// parseDurationAndCalculateEndTime parses duration string (e.g., "2:30") and calculates end time.
func (rs *ReservationService) parseDurationAndCalculateEndTime(startAt time.Time, duration string) (time.Time, error) {
	parts := strings.Split(duration, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("duration must be in HH:MM format")
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hours in duration")
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minutes in duration")
	}

	totalMinutes := hours*60 + minutes
	if totalMinutes <= 0 {
		return time.Time{}, fmt.Errorf("duration must be positive")
	}

	return startAt.Add(time.Duration(totalMinutes) * time.Minute), nil
}

// generateReservationCode generates a unique reservation code.
func (rs *ReservationService) generateReservationCode() string {
	// Simple implementation - in production, ensure uniqueness
	return fmt.Sprintf("RES%d", time.Now().Unix())
}

// checkForConflicts checks if a reservation conflicts with existing ones.
func (rs *ReservationService) checkForConflicts(ctx context.Context, reservation *model.Reservation) ([]*model.Reservation, error) {
	// Get all active reservations for the same restaurant and time period
	allReservations, err := rs.reservationRepo.GetByRestaurantID(ctx, reservation.RestaurantID, 1000, 0)
	if err != nil {
		return nil, err
	}

	var conflicts []*model.Reservation
	for _, existing := range allReservations {
		// Skip if different table
		if existing.TableID != reservation.TableID {
			continue
		}

		// Skip if not active
		if !existing.IsActive() {
			continue
		}

		// Skip if same reservation (for updates)
		if existing.ID == reservation.ID {
			continue
		}

		// Check for time overlap
		if reservation.IsOverlapping(existing) {
			conflicts = append(conflicts, existing)
		}
	}

	return conflicts, nil
}

// applyFilters applies additional filters to reservations.
func (rs *ReservationService) applyFilters(reservations []*model.Reservation, filters map[string]interface{}) []*model.Reservation {
	var filtered []*model.Reservation

	for _, reservation := range reservations {
		include := true

		// Filter by status
		if status, ok := filters["status"].(string); ok && status != "" {
			if string(reservation.Status) != status {
				include = false
			}
		}

		// Filter by table ID
		if tableIDStr, ok := filters["table_id"].(string); ok && tableIDStr != "" {
			if tableID, err := primitive.ObjectIDFromHex(tableIDStr); err == nil {
				if reservation.TableID != tableID {
					include = false
				}
			}
		}

		// Filter by date (start_at)
		if startDate, ok := filters["start_date"].(string); ok && startDate != "" {
			if date, err := time.Parse("2006-01-02", startDate); err == nil {
				reservationDate := time.Date(reservation.StartAt.Year(), reservation.StartAt.Month(), reservation.StartAt.Day(), 0, 0, 0, 0, reservation.StartAt.Location())
				if !reservationDate.Equal(date) {
					include = false
				}
			}
		}

		// Filter by is_seen
		if isSeen, ok := filters["is_seen"].(bool); ok {
			if reservation.IsSeen != isSeen {
				include = false
			}
		}

		if include {
			filtered = append(filtered, reservation)
		}
	}

	return filtered
}

// GetReservationByID retrieves a reservation by ID.
func (rs *ReservationService) GetReservationByID(ctx context.Context, id primitive.ObjectID) (*model.Reservation, error) {
	reservation, err := rs.reservationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}
	if reservation == nil {
		return nil, fmt.Errorf("reservation not found")
	}
	return reservation, nil
}

// UpdateReservationStatus updates the status of a reservation.
func (rs *ReservationService) UpdateReservationStatus(ctx context.Context, id primitive.ObjectID, req *model.UpdateReservationStatusRequest, employeeID primitive.ObjectID) (*model.Reservation, error) {
	reservation, err := rs.GetReservationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update status and related fields
	reservation.Status = req.Status
	reservation.UpdatedAt = time.Now()

	if req.AdminNotes != "" {
		reservation.AdminNotes = req.AdminNotes
	}

	if req.CancellationReason != "" {
		reservation.CancellationReason = req.CancellationReason
	}

	// Set timestamp based on status
	now := time.Now()
	switch req.Status {
	case model.StatusConfirmed:
		if reservation.ConfirmedAt == nil {
			reservation.ConfirmedAt = &now
		}
	case model.StatusArrived:
		if reservation.ArrivedAt == nil {
			reservation.ArrivedAt = &now
		}
	case model.StatusCompleted:
		if reservation.CompletedAt == nil {
			reservation.CompletedAt = &now
		}
	case model.StatusNoShow:
		if reservation.NoShowAt == nil {
			reservation.NoShowAt = &now
		}
	case model.StatusCanceledByAdmin:
		if reservation.CanceledAt == nil {
			reservation.CanceledAt = &now
		}
		reservation.CanceledByEmployeeID = &employeeID
	}

	// Update in database
	if err := rs.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	rs.logger.Info("Reservation status updated", "reservation_id", id.Hex(), "status", req.Status, "employee_id", employeeID.Hex())
	return reservation, nil
}

// UpdateReservationByAdmin updates a reservation by admin.
func (rs *ReservationService) UpdateReservationByAdmin(ctx context.Context, id primitive.ObjectID, req *model.UpdateReservationRequest, employeeID primitive.ObjectID) (*model.Reservation, error) {
	reservation, err := rs.GetReservationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update reservation fields
	if req.StartAt != nil {
		reservation.StartAt = *req.StartAt
		// Recalculate working day date
		reservation.WorkingDayDate = time.Date(req.StartAt.Year(), req.StartAt.Month(), req.StartAt.Day(), 0, 0, 0, 0, req.StartAt.Location())
	}
	if req.EndAt != nil {
		reservation.EndAt = *req.EndAt
	} else if req.Duration != nil {
		// Calculate end time from duration
		endAt, err := rs.parseDurationAndCalculateEndTime(reservation.StartAt, *req.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration format: %w", err)
		}
		reservation.EndAt = endAt
	}
	if req.TableID != nil {
		// Validate table exists
		restaurant, err := rs.restaurantRepo.GetByID(ctx, reservation.RestaurantID)
		if err != nil {
			return nil, fmt.Errorf("failed to get restaurant: %w", err)
		}
		room := restaurant.GetRoom(reservation.RoomID)
		if room == nil {
			return nil, fmt.Errorf("room not found")
		}
		table := room.GetTable(*req.TableID)
		if table == nil {
			return nil, fmt.Errorf("table not found")
		}
		if !table.IsEnabled {
			return nil, fmt.Errorf("table is not active")
		}
		reservation.TableID = *req.TableID
	}
	if req.GuestCount != nil {
		reservation.GuestCount = *req.GuestCount
	}
	if req.AdminNotes != nil {
		reservation.AdminNotes = *req.AdminNotes
	}
	if req.CancellationReason != nil {
		reservation.CancellationReason = *req.CancellationReason
	}

	reservation.UpdatedAt = time.Now()

	// Check for conflicts if time or table changed
	if req.StartAt != nil || req.EndAt != nil || req.Duration != nil || req.TableID != nil {
		conflicts, err := rs.checkForConflicts(ctx, reservation)
		if err != nil {
			return nil, fmt.Errorf("failed to check for conflicts: %w", err)
		}
		if len(conflicts) > 0 {
			return nil, fmt.Errorf("reservation conflicts with existing reservations at this time slot")
		}
	}

	// Update in database
	if err := rs.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	rs.logger.Info("Reservation updated by admin", "reservation_id", id.Hex(), "employee_id", employeeID.Hex())
	return reservation, nil
}

// CancelReservation cancels a reservation.
func (rs *ReservationService) CancelReservation(ctx context.Context, id primitive.ObjectID, reason string, employeeID primitive.ObjectID) (*model.Reservation, error) {
	reservation, err := rs.GetReservationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if reservation can be canceled
	if !reservation.IsCancelable() {
		return nil, fmt.Errorf("reservation cannot be canceled in current status: %s", reservation.Status)
	}

	// Update reservation status
	reservation.Status = model.StatusCanceledByAdmin
	reservation.CancellationReason = reason
	reservation.CanceledByEmployeeID = &employeeID
	now := time.Now()
	reservation.CanceledAt = &now
	reservation.UpdatedAt = now

	// Update in database
	if err := rs.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, fmt.Errorf("failed to cancel reservation: %w", err)
	}

	rs.logger.Info("Reservation canceled", "reservation_id", id.Hex(), "reason", reason, "employee_id", employeeID.Hex())
	return reservation, nil
}

// MarkReservationsAsSeen marks reservations as seen for a restaurant.
func (rs *ReservationService) MarkReservationsAsSeen(ctx context.Context, restaurantID primitive.ObjectID, req *model.MarkReservationsAsSeenRequest, employeeID primitive.ObjectID) error {
	// Validate restaurant exists
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return fmt.Errorf("restaurant not found")
	}

	if req.MarkAllUnseen {
		// Mark all unseen reservations for this restaurant as seen
		allReservations, err := rs.reservationRepo.GetByRestaurantID(ctx, restaurantID, 1000, 0)
		if err != nil {
			return fmt.Errorf("failed to get reservations: %w", err)
		}

		for _, reservation := range allReservations {
			if !reservation.IsSeen {
				reservation.IsSeen = true
				reservation.UpdatedAt = time.Now()
				if err := rs.reservationRepo.Update(ctx, reservation); err != nil {
					rs.logger.Error("Failed to mark reservation as seen", "reservation_id", reservation.ID.Hex(), "error", err)
					continue
				}
			}
		}

		rs.logger.Info("All unseen reservations marked as seen", "restaurant_id", restaurantID.Hex(), "employee_id", employeeID.Hex())
	} else if len(req.ReservationIDs) > 0 {
		// Mark specific reservations as seen
		for _, reservationIDStr := range req.ReservationIDs {
			reservationID, err := primitive.ObjectIDFromHex(reservationIDStr)
			if err != nil {
				rs.logger.Warn("Invalid reservation ID", "reservation_id", reservationIDStr)
				continue
			}

			reservation, err := rs.reservationRepo.GetByID(ctx, reservationID)
			if err != nil {
				rs.logger.Error("Failed to get reservation", "reservation_id", reservationIDStr, "error", err)
				continue
			}
			if reservation == nil {
				rs.logger.Warn("Reservation not found", "reservation_id", reservationIDStr)
				continue
			}

			// Verify reservation belongs to this restaurant
			if reservation.RestaurantID != restaurantID {
				rs.logger.Warn("Reservation does not belong to restaurant", "reservation_id", reservationIDStr, "restaurant_id", restaurantID.Hex())
				continue
			}

			reservation.IsSeen = true
			reservation.UpdatedAt = time.Now()
			if err := rs.reservationRepo.Update(ctx, reservation); err != nil {
				rs.logger.Error("Failed to mark reservation as seen", "reservation_id", reservationIDStr, "error", err)
				continue
			}
		}

		rs.logger.Info("Specific reservations marked as seen", "restaurant_id", restaurantID.Hex(), "count", len(req.ReservationIDs), "employee_id", employeeID.Hex())
	} else {
		return fmt.Errorf("either mark_all_unseen must be true or reservation_ids must be provided")
	}

	return nil
}

// GetReservationCounts gets reservation counts for a restaurant.
func (rs *ReservationService) GetReservationCounts(ctx context.Context, restaurantID primitive.ObjectID) (*model.ReservationCountsResponse, error) {
	// Validate restaurant exists
	restaurant, err := rs.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant: %w", err)
	}
	if restaurant == nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Get all reservations for the restaurant
	allReservations, err := rs.reservationRepo.GetByRestaurantID(ctx, restaurantID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations: %w", err)
	}

	// Count unseen and pending reservations
	unseenCount := 0
	pendingCount := 0

	for _, reservation := range allReservations {
		if !reservation.IsSeen {
			unseenCount++
		}
		if reservation.Status == model.StatusPending {
			pendingCount++
		}
	}

	return &model.ReservationCountsResponse{
		UnseenCount:  unseenCount,
		PendingCount: pendingCount,
	}, nil
}

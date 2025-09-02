// Package reservation contains the reservation model entities and logic.
package model

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status represents the status of a reservation.
type Status string

const (
	StatusBooking          Status = "BOOKING"
	StatusPending          Status = "PENDING"
	StatusConfirmed        Status = "CONFIRMED"
	StatusArrived          Status = "ARRIVED"
	StatusCompleted        Status = "COMPLETED"
	StatusNoShow           Status = "NO_SHOW"
	StatusCanceledByClient Status = "CANCELED_BY_CLIENT"
	StatusCanceledByAdmin  Status = "CANCELED_BY_ADMIN"
)

// AuthorizeMethod represents how the reservation was authorized.
type AuthorizeMethod string

const (
	AuthorizeMethodTelegram AuthorizeMethod = "TELEGRAM"
	AuthorizeMethodEmail    AuthorizeMethod = "EMAIL"
	AuthorizeMethodAdmin    AuthorizeMethod = "ADMIN"
)

// Reservation represents a table reservation.
type Reservation struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	Code         string              `json:"code" bson:"code"`
	RestaurantID primitive.ObjectID  `json:"restaurantId" bson:"restaurant_id"`
	RoomID       primitive.ObjectID  `json:"roomId" bson:"room_id"`
	TableID      primitive.ObjectID  `json:"tableId" bson:"table_id"`
	UserID       *primitive.ObjectID `json:"userId,omitempty" bson:"user_id,omitempty"`

	// Guest information
	FullName   string `json:"fullName" bson:"full_name"`
	Email      string `json:"email" bson:"email"`
	Phone      string `json:"phone" bson:"phone"`
	GuestCount int    `json:"guestCount" bson:"guest_count"`
	UserNotes  string `json:"userNotes" bson:"user_notes"`

	// Status and authorization
	Status          Status          `json:"status" bson:"status"`
	AuthorizeMethod AuthorizeMethod `json:"authorizeMethod" bson:"authorize_method"`
	IsUsedForAuth   bool            `json:"isUsedForAuth" bson:"is_used_for_auth"`

	// Time information
	StartAt        time.Time `json:"startAt" bson:"start_at"`
	EndAt          time.Time `json:"endAt" bson:"end_at"`
	InitialEndAt   time.Time `json:"initialEndAt" bson:"initial_end_at"`
	WorkingDayDate time.Time `json:"workingDayDate" bson:"working_day_date"`

	// Timestamps
	CreatedAt    time.Time  `json:"createdAt" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" bson:"updated_at"`
	AuthorizedAt *time.Time `json:"authorizedAt,omitempty" bson:"authorized_at,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmedAt,omitempty" bson:"confirmed_at,omitempty"`
	ArrivedAt    *time.Time `json:"arrivedAt,omitempty" bson:"arrived_at,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" bson:"completed_at,omitempty"`
	NoShowAt     *time.Time `json:"noShowAt,omitempty" bson:"no_show_at,omitempty"`
	CanceledAt   *time.Time `json:"canceledAt,omitempty" bson:"canceled_at,omitempty"`

	// Cancellation information
	CanceledByEmployeeID *primitive.ObjectID `json:"canceledByEmployeeId,omitempty" bson:"canceled_by_employee_id,omitempty"`
	CancellationReason   string              `json:"cancellationReason" bson:"cancellation_reason"`

	// Administrative fields
	AdminNotes string `json:"adminNotes" bson:"admin_notes"`
	IsSeen     bool   `json:"isSeen" bson:"is_seen"`

	// External integrations
	TgUsername        string `json:"tgUsername" bson:"tg_username"`
	ExternalBookingID string `json:"externalBookingId" bson:"external_booking_id"`
}

// CreateReservationRequest represents the request to create a reservation.
type CreateReservationRequest struct {
	TableID    primitive.ObjectID `json:"tableId" validate:"required"`
	Phone      string             `json:"phone,omitempty"`
	FullName   string             `json:"fullName" validate:"required,min=2,max=100"`
	GuestCount int                `json:"guestCount,omitempty" validate:"omitempty,min=1,max=20"`
	StartAt    time.Time          `json:"startAt" validate:"required"`
	Duration   string             `json:"duration" validate:"required"` // Format: "2:30" (2 hours 30 minutes)
	UserNotes  string             `json:"userNotes,omitempty" validate:"max=500"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CreateReservationRequest.
func (r *CreateReservationRequest) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with the same fields but StartAt as string
	type Alias CreateReservationRequest
	aux := &struct {
		StartAt string `json:"startAt"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse the StartAt time - just parse as UTC for now
	// The service layer will handle restaurant timezone conversion
	timeFormats := []string{
		time.RFC3339,                // 2025-09-01T10:00:00Z
		"2006-01-02T15:04:05",       // 2025-09-01T10:00:00
		"2006-01-02 15:04:05",       // 2025-09-01 10:00:00
		"2006-01-02T15:04:05.000Z",  // 2025-09-01T10:00:00.000Z
		"2006-01-02T15:04:05.000",   // 2025-09-01T10:00:00.000
	}

	var parsedTime time.Time
	var err error
	for _, format := range timeFormats {
		parsedTime, err = time.Parse(format, aux.StartAt)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("invalid startAt format: %s", aux.StartAt)
	}

	r.StartAt = parsedTime
	return nil
}

// UpdateReservationRequest represents the request to update a reservation.
type UpdateReservationRequest struct {
	StartAt            *time.Time          `json:"startAt,omitempty"`
	EndAt              *time.Time          `json:"endAt,omitempty"`
	Duration           *string             `json:"duration,omitempty"`
	TableID            *primitive.ObjectID `json:"tableId,omitempty"`
	GuestCount         *int                `json:"guestCount,omitempty" validate:"omitempty,min=1,max=20"`
	AdminNotes         *string             `json:"adminNotes,omitempty"`
	CancellationReason *string             `json:"cancellationReason,omitempty"`
}

// UpdateReservationStatusRequest represents the request to update reservation status.
type UpdateReservationStatusRequest struct {
	Status             Status `json:"status" validate:"required"`
	AdminNotes         string `json:"adminNotes"`
	CancellationReason string `json:"cancellationReason"`
}

// MarkReservationsAsSeenRequest represents the request to mark reservations as seen.
type MarkReservationsAsSeenRequest struct {
	ReservationIDs []string `json:"reservationIds,omitempty"`
	MarkAllUnseen  bool     `json:"markAllUnseen,omitempty"`
}

// ReservationCountsResponse represents reservation counts.
type ReservationCountsResponse struct {
	UnseenCount  int `json:"unseenCount"`
	PendingCount int `json:"pendingCount"`
}

// ReservationFilters represents filters for querying reservations.
type ReservationFilters struct {
	RestaurantID primitive.ObjectID `json:"restaurantId"`
	RoomID       primitive.ObjectID `json:"roomId,omitempty"`
	TableID      primitive.ObjectID `json:"tableId,omitempty"`
	StartAt      *time.Time         `json:"startAt,omitempty"`
	EndAt        *time.Time         `json:"endAt,omitempty"`
	Status       string             `json:"status,omitempty"`
	IsSeen       *bool              `json:"isSeen,omitempty"`
	Limit        int                `json:"limit,omitempty"`
	Skip         int                `json:"skip,omitempty"`
}

// Business logic methods

// IsActive returns true if the reservation is in an active state.
func (r *Reservation) IsActive() bool {
	return r.Status == StatusPending || r.Status == StatusConfirmed || r.Status == StatusArrived
}

// IsCancelable returns true if the reservation can be canceled.
func (r *Reservation) IsCancelable() bool {
	return r.Status == StatusPending || r.Status == StatusConfirmed
}

// IsFinished returns true if the reservation is in a final state.
func (r *Reservation) IsFinished() bool {
	return r.Status == StatusCompleted || r.Status == StatusNoShow ||
		r.Status == StatusCanceledByClient || r.Status == StatusCanceledByAdmin
}

// GetDurationMinutes returns the reservation duration in minutes.
func (r *Reservation) GetDurationMinutes() int {
	return int(r.EndAt.Sub(r.StartAt).Minutes())
}

// CanBeConfirmed returns true if the reservation can be confirmed.
func (r *Reservation) CanBeConfirmed() bool {
	return r.Status == StatusPending
}

// CanBeArrived returns true if the reservation can be marked as arrived.
func (r *Reservation) CanBeArrived() bool {
	return r.Status == StatusConfirmed
}

// CanBeCompleted returns true if the reservation can be marked as completed.
func (r *Reservation) CanBeCompleted() bool {
	return r.Status == StatusArrived
}

// CanBeNoShow returns true if the reservation can be marked as no-show.
func (r *Reservation) CanBeNoShow() bool {
	return r.Status == StatusPending || r.Status == StatusConfirmed
}

// IsOverlapping checks if this reservation overlaps with another reservation.
func (r *Reservation) IsOverlapping(other *Reservation) bool {
	// Same table is required for overlap
	if r.TableID != other.TableID {
		return false
	}

	// Check time overlap
	return r.StartAt.Before(other.EndAt) && r.EndAt.After(other.StartAt)
}

// GetTimeSlot returns a formatted time slot string.
func (r *Reservation) GetTimeSlot() string {
	return r.StartAt.Format("15:04") + " - " + r.EndAt.Format("15:04")
}

// IsExpired returns true if the reservation has expired (past end time).
func (r *Reservation) IsExpired() bool {
	return time.Now().After(r.EndAt)
}

// GetRemainingTime returns the remaining time until the reservation starts.
func (r *Reservation) GetRemainingTime() time.Duration {
	return time.Until(r.StartAt)
}

// IsStartingSoon returns true if the reservation is starting within the next hour.
func (r *Reservation) IsStartingSoon() bool {
	remaining := r.GetRemainingTime()
	return remaining > 0 && remaining <= time.Hour
}

// ReservationResponse represents the response when returning reservation data.
type ReservationResponse struct {
	ID           primitive.ObjectID  `json:"id"`
	Code         string              `json:"code"`
	RestaurantID primitive.ObjectID  `json:"restaurantId"`
	RoomID       primitive.ObjectID  `json:"roomId"`
	TableID      primitive.ObjectID  `json:"tableId"`
	UserID       *primitive.ObjectID `json:"userId,omitempty"`
	FullName     string              `json:"fullName"`
	Email        string              `json:"email"`
	Phone        string              `json:"phone"`
	GuestCount   int                 `json:"guestCount"`
	UserNotes    string              `json:"userNotes"`
	Status       Status              `json:"status"`
	StartAt      time.Time           `json:"startAt"`
	EndAt        time.Time           `json:"endAt"`
	Duration     int                 `json:"durationMinutes"`
	TimeSlot     string              `json:"timeSlot"`
	IsActive     bool                `json:"isActive"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

// ToResponse converts a Reservation to ReservationResponse.
func (r *Reservation) ToResponse() *ReservationResponse {
	return &ReservationResponse{
		ID:           r.ID,
		Code:         r.Code,
		RestaurantID: r.RestaurantID,
		RoomID:       r.RoomID,
		TableID:      r.TableID,
		UserID:       r.UserID,
		FullName:     r.FullName,
		Email:        r.Email,
		Phone:        r.Phone,
		GuestCount:   r.GuestCount,
		UserNotes:    r.UserNotes,
		Status:       r.Status,
		StartAt:      r.StartAt,
		EndAt:        r.EndAt,
		Duration:     r.GetDurationMinutes(),
		TimeSlot:     r.GetTimeSlot(),
		IsActive:     r.IsActive(),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

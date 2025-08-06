// Package reservation contains the reservation model entities and logic.
package model

import (
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
	RestaurantID primitive.ObjectID  `json:"restaurantId" bson:"restaurantId"`
	RoomID       primitive.ObjectID  `json:"room_id" bson:"room_id"`
	TableID      primitive.ObjectID  `json:"table_id" bson:"table_id"`
	UserID       *primitive.ObjectID `json:"user_id,omitempty" bson:"user_id,omitempty"`

	// Guest information
	FullName   string `json:"full_name" bson:"full_name"`
	Email      string `json:"email" bson:"email"`
	Phone      string `json:"phone" bson:"phone"`
	GuestCount int    `json:"guest_count" bson:"guest_count"`
	UserNotes  string `json:"user_notes" bson:"user_notes"`

	// Status and authorization
	Status          Status          `json:"status" bson:"status"`
	AuthorizeMethod AuthorizeMethod `json:"authorize_method" bson:"authorize_method"`
	IsUsedForAuth   bool            `json:"is_used_for_auth" bson:"is_used_for_auth"`

	// Time information
	StartAt        time.Time `json:"start_at" bson:"start_at"`
	EndAt          time.Time `json:"end_at" bson:"end_at"`
	InitialEndAt   time.Time `json:"initial_end_at" bson:"initial_end_at"`
	WorkingDayDate time.Time `json:"working_day_date" bson:"working_day_date"`

	// Timestamps
	CreatedAt    time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" bson:"updated_at"`
	AuthorizedAt *time.Time `json:"authorized_at,omitempty" bson:"authorized_at,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty" bson:"confirmed_at,omitempty"`
	ArrivedAt    *time.Time `json:"arrived_at,omitempty" bson:"arrived_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	NoShowAt     *time.Time `json:"no_show_at,omitempty" bson:"no_show_at,omitempty"`
	CanceledAt   *time.Time `json:"canceled_at,omitempty" bson:"canceled_at,omitempty"`

	// Cancellation information
	CanceledByUserID   *primitive.ObjectID `json:"canceled_by_user_id,omitempty" bson:"canceled_by_user_id,omitempty"`
	CancellationReason string              `json:"cancellation_reason" bson:"cancellation_reason"`

	// Administrative fields
	AdminNotes string `json:"admin_notes" bson:"admin_notes"`
	IsSeen     bool   `json:"is_seen" bson:"is_seen"`

	// External integrations
	TgUsername        string `json:"tg_username" bson:"tg_username"`
	ExternalBookingID string `json:"external_booking_id" bson:"external_booking_id"`
}

// CreateReservationRequest represents the request to create a reservation.
type CreateReservationRequest struct {
	TableID    primitive.ObjectID `json:"table_id" validate:"required"`
	FullName   string    `json:"full_name" validate:"required,min=2,max=100"`
	Email      string    `json:"email" validate:"required,email"`
	Phone      string    `json:"phone" validate:"required"`
	GuestCount int       `json:"guest_count" validate:"required,min=1,max=20"`
	StartAt    time.Time `json:"start_at" validate:"required"`
	Duration   string    `json:"duration" validate:"required"` // Format: "2:30" (2 hours 30 minutes)
	UserNotes  string    `json:"user_notes" validate:"max=500"`
}

// UpdateReservationRequest represents the request to update a reservation.
type UpdateReservationRequest struct {
	StartAt            *time.Time `json:"start_at,omitempty"`
	EndAt              *time.Time `json:"end_at,omitempty"`
	Duration           *string    `json:"duration,omitempty"`
	TableID            *primitive.ObjectID `json:"table_id,omitempty"`
	GuestCount         *int       `json:"guest_count,omitempty" validate:"omitempty,min=1,max=20"`
	AdminNotes         *string    `json:"admin_notes,omitempty"`
	CancellationReason *string    `json:"cancellation_reason,omitempty"`
}

// UpdateReservationStatusRequest represents the request to update reservation status.
type UpdateReservationStatusRequest struct {
	Status             Status `json:"status" validate:"required"`
	AdminNotes         string `json:"admin_notes"`
	CancellationReason string `json:"cancellation_reason"`
}

// MarkReservationsAsSeenRequest represents the request to mark reservations as seen.
type MarkReservationsAsSeenRequest struct {
	ReservationIDs []string `json:"reservation_ids,omitempty"`
	MarkAllUnseen  bool     `json:"mark_all_unseen,omitempty"`
}

// ReservationCountsResponse represents reservation counts.
type ReservationCountsResponse struct {
	UnseenCount  int `json:"unseen_count"`
	PendingCount int `json:"pending_count"`
}

// ReservationFilters represents filters for querying reservations.
type ReservationFilters struct {
	RestaurantID primitive.ObjectID `json:"restaurantId"`
	RoomID       primitive.ObjectID `json:"room_id,omitempty"`
	TableID      primitive.ObjectID `json:"table_id,omitempty"`
	StartAt      *time.Time         `json:"start_at,omitempty"`
	EndAt        *time.Time         `json:"end_at,omitempty"`
	Status       string             `json:"status,omitempty"`
	IsSeen       *bool              `json:"is_seen,omitempty"`
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
	RoomID       primitive.ObjectID  `json:"room_id"`
	TableID      primitive.ObjectID  `json:"table_id"` 
	UserID       *primitive.ObjectID `json:"user_id,omitempty"`
	FullName     string              `json:"full_name"`
	Email        string              `json:"email"`
	Phone        string              `json:"phone"`
	GuestCount   int                 `json:"guest_count"`
	UserNotes    string              `json:"user_notes"`
	Status       Status              `json:"status"`
	StartAt      time.Time           `json:"start_at"`
	EndAt        time.Time           `json:"end_at"`
	Duration     int                 `json:"duration_minutes"`
	TimeSlot     string              `json:"time_slot"`
	IsActive     bool                `json:"is_active"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
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

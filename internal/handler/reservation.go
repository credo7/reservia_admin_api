package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/service"
	"github.com/reservia/api/pkg/logger"
)

type ReservationHandler struct {
	reservationService *service.ReservationService
	authService        *service.AuthService
	logger             logger.Logger
}

func NewReservationHandler(reservationService *service.ReservationService, authService *service.AuthService, logger logger.Logger) *ReservationHandler {
	return &ReservationHandler{
		reservationService: reservationService,
		authService:        authService,
		logger:             logger,
	}
}

// CreateReservationByEmployee handles POST /restaurants/{restaurantId}/rooms/{roomId}/reservations/by_employee.
//
//	@Summary		Create a reservation by employee
//	@Description	Create a new reservation by an admin employee
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string							true	"Restaurant ID"
//	@Param			roomId			path		string							true	"Room ID"
//	@Param			reservation		body		model.CreateReservationRequest	true	"Reservation creation data"
//	@Success		201				{object}	model.ReservationResponse		"Reservation created successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Restaurant or room not found"
//	@Failure		409				{object}	map[string]string				"Reservation conflict"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/reservations/by_employee [post]
//	@Security		BearerAuth
func (h *ReservationHandler) CreateReservationByEmployee(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation creation by employee requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var req model.CreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reservation, err := h.reservationService.CreateReservationByEmployee(r.Context(), restaurantID, roomID, &req, employeeID)
	if err != nil {
		h.logger.Error("Failed to create reservation", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else if strings.Contains(err.Error(), "conflict") {
			h.writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "not active") {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else if strings.Contains(err.Error(), "invalid duration") {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to create reservation")
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, reservation.ToResponse())
}

// GetReservationsByRoom handles GET /restaurants/{restaurantId}/rooms/{roomId}/reservations.
//
//	@Summary		Get reservations for a room
//	@Description	Retrieve reservations for a specific room with optional filtering
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId		path		string	true	"Restaurant ID"
//	@Param			roomId				path		string	true	"Room ID"
//	@Param			limit				query		int		false	"Number of reservations to return (default: 50)"
//	@Param			offset				query		int		false	"Number of reservations to skip (default: 0)"
//	@Param			status				query		string	false	"Filter by status"
//	@Param			table_id			query		string	false	"Filter by table ID"
//	@Param			start_date			query		string	false	"Filter by date (YYYY-MM-DD format)"
//	@Param			is_seen				query		bool	false	"Filter by seen status"
//	@Success		200					{object}	map[string]interface{}	"Reservations retrieved successfully"
//	@Failure		400					{object}	map[string]string		"Invalid request"
//	@Failure		404					{object}	map[string]string		"Restaurant or room not found"
//	@Failure		500					{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/reservations [get]
//	@Security		BearerAuth
func (h *ReservationHandler) GetReservationsByRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room reservations retrieval requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	status := r.URL.Query().Get("status")
	tableID := r.URL.Query().Get("table_id")
	startDate := r.URL.Query().Get("start_date")
	isSeenStr := r.URL.Query().Get("is_seen")

	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build filters
	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}
	if tableID != "" {
		filters["table_id"] = tableID
	}
	if startDate != "" {
		filters["start_date"] = startDate
	}
	if isSeenStr != "" {
		if isSeen, err := strconv.ParseBool(isSeenStr); err == nil {
			filters["is_seen"] = isSeen
		}
	}

	reservations, err := h.reservationService.GetReservationsByRoom(r.Context(), restaurantID, roomID, filters, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get room reservations", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to get reservations")
		}
		return
	}

	// Convert to response format
	responses := make([]*model.ReservationResponse, len(reservations))
	for i, reservation := range reservations {
		responses[i] = reservation.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"reservations": responses,
		"limit":        limit,
		"offset":       offset,
		"count":        len(responses),
		"filters":      filters,
	})
}

// GetReservation handles GET /reservations/{reservationId}.
//
//	@Summary		Get reservation by ID
//	@Description	Retrieve a specific reservation by its ID
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			reservationId	path		string						true	"Reservation ID"
//	@Success		200				{object}	model.ReservationResponse	"Reservation retrieved successfully"
//	@Failure		400				{object}	map[string]string			"Invalid request"
//	@Failure		404				{object}	map[string]string			"Reservation not found"
//	@Failure		500				{object}	map[string]string			"Internal server error"
//	@Router			/reservations/{reservationId} [get]
//	@Security		BearerAuth
func (h *ReservationHandler) GetReservation(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation retrieval requested", "employee_id", employeeID)

	reservationIDStr := chi.URLParam(r, "reservationId")
	reservationID, err := primitive.ObjectIDFromHex(reservationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid reservation ID")
		return
	}

	reservation, err := h.reservationService.GetReservationByID(r.Context(), reservationID)
	if err != nil {
		h.logger.Error("Failed to get reservation", "reservation_id", reservationID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Reservation not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to get reservation")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, reservation.ToResponse())
}

// UpdateReservationByAdmin handles PATCH /reservations/{reservationId}/by_admin.
//
//	@Summary		Update reservation by admin
//	@Description	Update a reservation by an admin employee
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			reservationId	path		string							true	"Reservation ID"
//	@Param			reservation		body		model.UpdateReservationRequest	true	"Reservation update data"
//	@Success		200				{object}	model.ReservationResponse		"Reservation updated successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Reservation not found"
//	@Failure		409				{object}	map[string]string				"Reservation conflict"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/reservations/{reservationId}/by_admin [patch]
//	@Security		BearerAuth
func (h *ReservationHandler) UpdateReservationByAdmin(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation update by admin requested", "employee_id", employeeID)

	reservationIDStr := chi.URLParam(r, "reservationId")
	reservationID, err := primitive.ObjectIDFromHex(reservationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid reservation ID")
		return
	}

	var req model.UpdateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reservation, err := h.reservationService.UpdateReservationByAdmin(r.Context(), reservationID, &req, employeeID)
	if err != nil {
		h.logger.Error("Failed to update reservation", "reservation_id", reservationID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else if strings.Contains(err.Error(), "conflict") {
			h.writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "not active") {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else if strings.Contains(err.Error(), "invalid duration") {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update reservation")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, reservation.ToResponse())
}

// UpdateReservationStatus handles PATCH /reservations/{reservationId}/status.
//
//	@Summary		Update reservation status
//	@Description	Update the status of a reservation
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			reservationId	path		string									true	"Reservation ID"
//	@Param			status			body		model.UpdateReservationStatusRequest	true	"Status update data"
//	@Success		200				{object}	model.ReservationResponse				"Status updated successfully"
//	@Failure		400				{object}	map[string]string						"Invalid request"
//	@Failure		404				{object}	map[string]string						"Reservation not found"
//	@Failure		500				{object}	map[string]string						"Internal server error"
//	@Router			/reservations/{reservationId}/status [patch]
//	@Security		BearerAuth
func (h *ReservationHandler) UpdateReservationStatus(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation status update requested", "employee_id", employeeID)

	reservationIDStr := chi.URLParam(r, "reservationId")
	reservationID, err := primitive.ObjectIDFromHex(reservationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid reservation ID")
		return
	}

	var req model.UpdateReservationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reservation, err := h.reservationService.UpdateReservationStatus(r.Context(), reservationID, &req, employeeID)
	if err != nil {
		h.logger.Error("Failed to update reservation status", "reservation_id", reservationID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Reservation not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update reservation status")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, reservation.ToResponse())
}

// CancelReservation handles POST /reservations/{reservationId}/cancel.
//
//	@Summary		Cancel reservation
//	@Description	Cancel a reservation with reason
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			reservationId	path		string						true	"Reservation ID"
//	@Param			cancelData		body		map[string]string		true	"Cancellation data with reason"
//	@Success		200				{object}	model.ReservationResponse	"Reservation canceled successfully"
//	@Failure		400				{object}	map[string]string			"Invalid request"
//	@Failure		404				{object}	map[string]string			"Reservation not found"
//	@Failure		500				{object}	map[string]string			"Internal server error"
//	@Router			/reservations/{reservationId}/cancel [post]
//	@Security		BearerAuth
func (h *ReservationHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation cancellation requested", "employee_id", employeeID)

	reservationIDStr := chi.URLParam(r, "reservationId")
	reservationID, err := primitive.ObjectIDFromHex(reservationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid reservation ID")
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reason, ok := req["reason"]
	if !ok {
		reason = "Canceled by admin"
	}

	reservation, err := h.reservationService.CancelReservation(r.Context(), reservationID, reason, employeeID)
	if err != nil {
		h.logger.Error("Failed to cancel reservation", "reservation_id", reservationID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Reservation not found")
		} else if strings.Contains(err.Error(), "cannot be canceled") {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to cancel reservation")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, reservation.ToResponse())
}

// Helper methods

// authenticateRequest extracts and validates the Bearer token, returning the employee ID.
func (h *ReservationHandler) authenticateRequest(r *http.Request) (primitive.ObjectID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return primitive.NilObjectID, fmt.Errorf("authorization header is required")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return primitive.NilObjectID, fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return primitive.NilObjectID, fmt.Errorf("token is required")
	}

	employeeID, err := h.authService.ValidateToken(token)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid or expired token")
	}

	return employeeID, nil
}

func (h *ReservationHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

func (h *ReservationHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

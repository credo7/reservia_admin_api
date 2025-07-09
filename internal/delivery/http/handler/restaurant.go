// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/domain/restaurant"
	restaurantUseCase "github.com/reservia/api/internal/usecase/restaurant"
	"github.com/reservia/api/pkg/logger"
)

// RestaurantHandler handles restaurant-related HTTP requests.
type RestaurantHandler struct {
	restaurantUseCase *restaurantUseCase.UseCase
	logger            logger.Logger
}

// NewRestaurantHandler creates a new restaurant handler.
func NewRestaurantHandler(restaurantUseCase *restaurantUseCase.UseCase, logger logger.Logger) *RestaurantHandler {
	return &RestaurantHandler{
		restaurantUseCase: restaurantUseCase,
		logger:            logger,
	}
}

// CreateRestaurant handles POST /restaurants.
//
//	@Summary		Create a new restaurant
//	@Description	Create a new restaurant with the provided information
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurant	body		restaurant.CreateRestaurantRequest	true	"Restaurant creation data"
//	@Success		201			{object}	restaurant.RestaurantResponse		"Restaurant created successfully"
//	@Failure		400			{object}	map[string]string					"Invalid request body"
//	@Failure		500			{object}	map[string]string					"Internal server error"
//	@Router			/restaurants [post]
func (h *RestaurantHandler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	var req restaurant.CreateRestaurantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	createdRestaurant, err := h.restaurantUseCase.CreateRestaurant(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create restaurant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to create restaurant")
		return
	}

	h.writeJSON(w, http.StatusCreated, createdRestaurant.ToResponse())
}

// GetRestaurant handles GET /restaurants/{id}.
//
//	@Summary		Get restaurant by ID
//	@Description	Retrieve a restaurant by its unique ID
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string						true	"Restaurant ID"
//	@Success		200	{object}	restaurant.RestaurantResponse	"Restaurant found"
//	@Failure		400	{object}	map[string]string			"Invalid restaurant ID"
//	@Failure		404	{object}	map[string]string			"Restaurant not found"
//	@Router			/restaurants/{id} [get]
func (h *RestaurantHandler) GetRestaurant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	rest, err := h.restaurantUseCase.GetRestaurantByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get restaurant", "error", err)
		h.writeError(w, http.StatusNotFound, "Restaurant not found")
		return
	}

	h.writeJSON(w, http.StatusOK, rest.ToResponse())
}

// UpdateRestaurant handles PUT /restaurants/{id}.
func (h *RestaurantHandler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req restaurant.UpdateRestaurantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedRestaurant, err := h.restaurantUseCase.UpdateRestaurant(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update restaurant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update restaurant")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedRestaurant.ToResponse())
}

// DeleteRestaurant handles DELETE /restaurants/{id}.
func (h *RestaurantHandler) DeleteRestaurant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	if err := h.restaurantUseCase.DeleteRestaurant(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete restaurant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to delete restaurant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListRestaurants handles GET /restaurants.
//
//	@Summary		List restaurants
//	@Description	Retrieve a list of restaurants with filtering and pagination
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			limit		query		int						false	"Number of restaurants to return (default: 10)"
//	@Param			offset		query		int						false	"Number of restaurants to skip (default: 0)"
//	@Param			is_active	query		bool					false	"Filter by active status"
//	@Success		200			{object}	map[string]interface{}	"List of restaurants with pagination info"
//	@Failure		500			{object}	map[string]string		"Internal server error"
//	@Router			/restaurants [get]
func (h *RestaurantHandler) ListRestaurants(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	isActive := r.URL.Query().Get("is_active")

	limit := 10 // default
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
	if isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			filters["is_active"] = active
		}
	}

	restaurants, err := h.restaurantUseCase.ListRestaurants(r.Context(), filters, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list restaurants", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list restaurants")
		return
	}

	// Convert to response format
	responses := make([]*restaurant.RestaurantResponse, len(restaurants))
	for i, rest := range restaurants {
		responses[i] = rest.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"restaurants": responses,
		"limit":       limit,
		"offset":      offset,
		"count":       len(responses),
	})
}

// GetRestaurantRooms handles GET /restaurants/{id}/rooms.
func (h *RestaurantHandler) GetRestaurantRooms(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	rooms, err := h.restaurantUseCase.GetRestaurantRooms(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get restaurant rooms", "error", err)
		h.writeError(w, http.StatusNotFound, "Restaurant not found")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"rooms": rooms,
		"count": len(rooms),
	})
}

// GetRestaurantRoom handles GET /restaurants/{id}/rooms/{roomId}.
func (h *RestaurantHandler) GetRestaurantRoom(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	roomID := chi.URLParam(r, "roomId")
	room, err := h.restaurantUseCase.GetRestaurantRoom(r.Context(), id, roomID)
	if err != nil {
		h.logger.Error("Failed to get restaurant room", "error", err)
		h.writeError(w, http.StatusNotFound, "Room not found")
		return
	}

	h.writeJSON(w, http.StatusOK, room)
}

// Helper methods

func (h *RestaurantHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *RestaurantHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

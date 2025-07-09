package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reservia/api/internal/domain/restaurant"
	restaurantUseCase "github.com/reservia/api/internal/usecase/restaurant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoomHandler struct {
	restaurantUseCase *restaurantUseCase.UseCase
}

func NewRoomHandler(restaurantUseCase *restaurantUseCase.UseCase) *RoomHandler {
	return &RoomHandler{
		restaurantUseCase: restaurantUseCase,
	}
}

// GetRooms godoc
// @Summary      Get all rooms for a restaurant
// @Description  Retrieve all rooms for a specific restaurant
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        restaurantID path string true "Restaurant ID"
// @Success      200  {array}   restaurant.Room
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/restaurants/{restaurantID}/rooms [get]
func (h *RoomHandler) GetRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	restaurantIDStr := chi.URLParam(r, "restaurantID")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		http.Error(w, "Invalid restaurant ID", http.StatusBadRequest)
		return
	}

	rooms, err := h.restaurantUseCase.GetRestaurantRooms(ctx, restaurantID)
	if err != nil {
		http.Error(w, "Failed to get rooms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rooms)
}

// GetRoom godoc
// @Summary      Get a specific room
// @Description  Retrieve a specific room by ID from a restaurant
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        restaurantID path string true "Restaurant ID"
// @Param        roomID path string true "Room ID"
// @Success      200  {object}  restaurant.Room
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/restaurants/{restaurantID}/rooms/{roomID} [get]
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	restaurantIDStr := chi.URLParam(r, "restaurantID")
	roomIDStr := chi.URLParam(r, "roomID")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		http.Error(w, "Invalid restaurant ID", http.StatusBadRequest)
		return
	}

	room, err := h.restaurantUseCase.GetRestaurantRoom(ctx, restaurantID, roomIDStr)
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(room)
}

// CreateRoom godoc
// @Summary      Create a new room
// @Description  Create a new room in a restaurant (placeholder implementation)
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        restaurantID path string true "Restaurant ID"
// @Param        request body restaurant.Room true "Room creation request"
// @Success      201  {object}  restaurant.Room
// @Failure      400  {object}  ErrorResponse
// @Failure      501  {object}  ErrorResponse
// @Router       /api/v1/restaurants/{restaurantID}/rooms [post]
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var room restaurant.Room
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement room creation
	room.ID = primitive.NewObjectID().Hex()

	response := map[string]interface{}{
		"message": "Room creation not yet implemented",
		"room":    room,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

// UpdateRoom godoc
// @Summary      Update a room
// @Description  Update a specific room in a restaurant (placeholder implementation)
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        restaurantID path string true "Restaurant ID"
// @Param        roomID path string true "Room ID"
// @Param        request body restaurant.Room true "Room update request"
// @Success      200  {object}  restaurant.Room
// @Failure      400  {object}  ErrorResponse
// @Failure      501  {object}  ErrorResponse
// @Router       /api/v1/restaurants/{restaurantID}/rooms/{roomID} [patch]
func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	var room restaurant.Room
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message": "Room update not yet implemented",
		"room":    room,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

// DeleteRoom godoc
// @Summary      Delete a room
// @Description  Delete a specific room from a restaurant (placeholder implementation)
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        restaurantID path string true "Restaurant ID"
// @Param        roomID path string true "Room ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      501  {object}  ErrorResponse
// @Router       /api/v1/restaurants/{restaurantID}/rooms/{roomID} [delete]
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "roomID")

	response := map[string]string{
		"message": "Room deletion not yet implemented",
		"room_id": roomIDStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(response)
}

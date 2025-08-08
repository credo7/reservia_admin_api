package handler

import (
	"encoding/json"
	"fmt"
	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/service"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/pkg/logger"
)

type RoomHandler struct {
	restaurantService *service.RestaurantService
	authService       *service.AuthService
	logger            logger.Logger
}

func NewRoomHandler(restaurantService *service.RestaurantService, authService *service.AuthService, logger logger.Logger) *RoomHandler {
	return &RoomHandler{
		restaurantService: restaurantService,
		authService:       authService,
		logger:            logger,
	}
}

// GetRooms handles GET /restaurants/{restaurantId}/rooms.
//
//	@Summary		Get all rooms for a restaurant
//	@Description	Retrieve all rooms for a specific restaurant
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Success		200				{array}		model.Room				"Rooms retrieved successfully"
//	@Failure		400				{object}	map[string]string		"Invalid request"
//	@Failure		404				{object}	map[string]string		"Restaurant not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms [get]
//	@Security		BearerAuth
func (h *RoomHandler) GetRooms(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Rooms retrieval requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	rooms, err := h.restaurantService.GetRestaurantRooms(r.Context(), restaurantID)
	if err != nil {
		h.logger.Error("Failed to get rooms", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Restaurant not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to get rooms")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, rooms)
}

// GetRoom handles GET /restaurants/{restaurantId}/rooms/{roomId}.
//
//	@Summary		Get a specific room
//	@Description	Retrieve a specific room by ID from a restaurant
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Param			roomId			path		string					true	"Room ID"
//	@Success		200				{object}	model.Room				"Room retrieved successfully"
//	@Failure		400				{object}	map[string]string		"Invalid request"
//	@Failure		404				{object}	map[string]string		"Room not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId} [get]
//	@Security		BearerAuth
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room retrieval requested", "employee_id", employeeID)

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

	room, err := h.restaurantService.GetRestaurantRoom(r.Context(), restaurantID, roomID)
	if err != nil {
		h.logger.Error("Failed to get room", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to get room")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, room)
}

// CreateRoom handles POST /restaurants/{restaurantId}/rooms.
//
//	@Summary		Create a new room
//	@Description	Create a new room in a restaurant
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Param			room			body		model.CreateRoomRequest	true	"Room creation data"
//	@Success		201				{object}	model.Room				"Room created successfully"
//	@Failure		400				{object}	map[string]string		"Invalid request"
//	@Failure		409				{object}	map[string]string		"Room name already exists"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms [post]
//	@Security		BearerAuth
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room creation requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedRestaurant, err := h.restaurantService.AddRoom(r.Context(), restaurantID, &req)
	if err != nil {
		h.logger.Error("Failed to create room", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Restaurant not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to create room")
		}
		return
	}

	// Return the newly created room
	var createdRoom *model.Room
	for _, room := range updatedRestaurant.Rooms {
		if room.Name == req.Name {
			createdRoom = &room
			break
		}
	}

	h.writeJSON(w, http.StatusCreated, createdRoom)
}

// UpdateRoom handles PATCH /restaurants/{restaurantId}/rooms/{roomId}.
//
//	@Summary		Update a room
//	@Description	Update a specific room in a restaurant
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string						true	"Restaurant ID"
//	@Param			roomId			path		string						true	"Room ID"
//	@Param			room			body		model.UpdateRoomRequest		true	"Room update data"
//	@Success		200				{object}	model.Room					"Room updated successfully"
//	@Failure		400				{object}	map[string]string			"Invalid request"
//	@Failure		404				{object}	map[string]string			"Room not found"
//	@Failure		409				{object}	map[string]string			"Room name already exists"
//	@Failure		500				{object}	map[string]string			"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId} [patch]
//	@Security		BearerAuth
func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room update requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.UpdateRoom(r.Context(), restaurantID, roomID, &req)
	if err != nil {
		h.logger.Error("Failed to update room", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update room")
		}
		return
	}

	// Return the updated room
	var updatedRoom *model.Room
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			updatedRoom = &room
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedRoom)
}

// DeleteRoom handles DELETE /restaurants/{restaurantId}/rooms/{roomId}.
//
//	@Summary		Delete a room
//	@Description	Delete a specific room from a restaurant
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Param			roomId			path		string					true	"Room ID"
//	@Success		200				{object}	map[string]string		"Room deleted successfully"
//	@Failure		400				{object}	map[string]string		"Invalid request"
//	@Failure		404				{object}	map[string]string		"Room not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId} [delete]
//	@Security		BearerAuth
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room deletion requested", "employee_id", employeeID)

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

	_, err = h.restaurantService.DeleteRoom(r.Context(), restaurantID, roomID)
	if err != nil {
		h.logger.Error("Failed to delete room", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to delete room")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Room deleted successfully",
		"room_id": roomIDStr,
	})
}

// EnableRoom handles POST /restaurants/{restaurantId}/rooms/{roomId}/enable.
//
//	@Summary		Enable a room
//	@Description	Enable a room (set isEnabled to true)
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string				true	"Restaurant ID"
//	@Param			roomId			path		string				true	"Room ID"
//	@Success		200				{object}	model.Room			"Room enabled successfully"
//	@Failure		400				{object}	map[string]string	"Invalid request"
//	@Failure		404				{object}	map[string]string	"Room not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/enable [post]
//	@Security		BearerAuth
func (h *RoomHandler) EnableRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room enable requested", "employee_id", employeeID)

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

	updatedRestaurant, err := h.restaurantService.EnableRoom(r.Context(), restaurantID, roomID)
	if err != nil {
		h.logger.Error("Failed to enable room", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to enable room")
		}
		return
	}

	// Return the updated room
	var updatedRoom *model.Room
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			updatedRoom = &room
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedRoom)
}

// DisableRoom handles POST /restaurants/{restaurantId}/rooms/{roomId}/disable.
//
//	@Summary		Disable a room
//	@Description	Disable a room (set isEnabled to false)
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string				true	"Restaurant ID"
//	@Param			roomId			path		string				true	"Room ID"
//	@Success		200				{object}	model.Room			"Room disabled successfully"
//	@Failure		400				{object}	map[string]string	"Invalid request"
//	@Failure		404				{object}	map[string]string	"Room not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/disable [post]
//	@Security		BearerAuth
func (h *RoomHandler) DisableRoom(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Room disable requested", "employee_id", employeeID)

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

	updatedRestaurant, err := h.restaurantService.DisableRoom(r.Context(), restaurantID, roomID)
	if err != nil {
		h.logger.Error("Failed to disable room", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to disable room")
		}
		return
	}

	// Return the updated room
	var updatedRoom *model.Room
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			updatedRoom = &room
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedRoom)
}

// SaveElements handles POST /restaurants/{restaurantId}/rooms/{roomId}/elements.
//
//	@Summary		Save multiple elements to a room
//	@Description	Save multiple elements to a room (replaces all existing elements)
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string							true	"Restaurant ID"
//	@Param			roomId			path		string							true	"Room ID"
//	@Param			elements		body		model.SaveElementsRequest		true	"Elements to save"
//	@Success		200				{object}	map[string]interface{}			"Elements saved successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Room not found"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/elements [post]
//	@Security		BearerAuth
func (h *RoomHandler) SaveElements(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Elements save requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.SaveElementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.SaveElements(r.Context(), restaurantID, roomID, &req)
	if err != nil {
		h.logger.Error("Failed to save elements", "restaurant_id", restaurantID, "room_id", roomIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to save elements")
		}
		return
	}

	// Return the updated room with elements
	var updatedRoom *model.Room
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			updatedRoom = &room
			break
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Elements saved successfully",
		"elements": updatedRoom.Elements,
		"count":    len(updatedRoom.Elements),
	})
}

// UpdateElement handles PATCH /restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId}.
//
//	@Summary		Update an element
//	@Description	Update a specific element in a room
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string							true	"Restaurant ID"
//	@Param			roomId			path		string							true	"Room ID"
//	@Param			elementId		path		string							true	"Element ID"
//	@Param			element			body		model.UpdateElementRequest		true	"Element update data"
//	@Success		200				{object}	model.Element					"Element updated successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Element not found"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId} [patch]
//	@Security		BearerAuth
func (h *RoomHandler) UpdateElement(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Element update requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")
	elementIDStr := chi.URLParam(r, "elementId")

	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.UpdateElementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	elementID, err := primitive.ObjectIDFromHex(elementIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid element ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.UpdateElement(r.Context(), restaurantID, roomID, elementID, &req)
	if err != nil {
		h.logger.Error("Failed to update element", "restaurant_id", restaurantID, "room_id", roomIDStr, "element_id", elementIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update element")
		}
		return
	}

	// Return the updated element
	var updatedElement *model.Element
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			for _, element := range room.Elements {
				if element.ID == elementID {
					updatedElement = &element
					break
				}
			}
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedElement)
}

// EnableElement handles PATCH /restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId}/enable.
//
//	@Summary		Enable an element
//	@Description	Enable a specific element in a room
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string				true	"Restaurant ID"
//	@Param			roomId			path		string				true	"Room ID"
//	@Param			elementId		path		string				true	"Element ID"
//	@Success		200				{object}	model.Element		"Element enabled successfully"
//	@Failure		400				{object}	map[string]string	"Invalid request"
//	@Failure		404				{object}	map[string]string	"Element not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId}/enable [patch]
//	@Security		BearerAuth
func (h *RoomHandler) EnableElement(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Element enable requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")
	elementIDStr := chi.URLParam(r, "elementId")

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

	elementID, err := primitive.ObjectIDFromHex(elementIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid element ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.EnableElement(r.Context(), restaurantID, roomID, elementID)
	if err != nil {
		h.logger.Error("Failed to enable element", "restaurant_id", restaurantID, "room_id", roomIDStr, "element_id", elementIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to enable element")
		}
		return
	}

	// Return the updated element
	var updatedElement *model.Element
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			for _, element := range room.Elements {
				if element.ID == elementID {
					updatedElement = &element
					break
				}
			}
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedElement)
}

// DisableElement handles PATCH /restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId}/disable.
//
//	@Summary		Disable an element
//	@Description	Disable a specific element in a room
//	@Tags			rooms
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string				true	"Restaurant ID"
//	@Param			roomId			path		string				true	"Room ID"
//	@Param			elementId		path		string				true	"Element ID"
//	@Success		200				{object}	model.Element		"Element disabled successfully"
//	@Failure		400				{object}	map[string]string	"Invalid request"
//	@Failure		404				{object}	map[string]string	"Element not found"
//	@Failure		500				{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{restaurantId}/rooms/{roomId}/elements/{elementId}/disable [patch]
//	@Security		BearerAuth
func (h *RoomHandler) DisableElement(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Element disable requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	roomIDStr := chi.URLParam(r, "roomId")
	elementIDStr := chi.URLParam(r, "elementId")

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

	elementID, err := primitive.ObjectIDFromHex(elementIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid element ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.DisableElement(r.Context(), restaurantID, roomID, elementID)
	if err != nil {
		h.logger.Error("Failed to disable element", "restaurant_id", restaurantID, "room_id", roomIDStr, "element_id", elementIDStr, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to disable element")
		}
		return
	}

	// Return the updated element
	var updatedElement *model.Element
	for _, room := range updatedRestaurant.Rooms {
		if room.ID == roomID {
			for _, element := range room.Elements {
				if element.ID == elementID {
					updatedElement = &element
					break
				}
			}
			break
		}
	}

	h.writeJSON(w, http.StatusOK, updatedElement)
}

// Helper methods

// authenticateRequest extracts and validates the Bearer token, returning the employee ID.
func (h *RoomHandler) authenticateRequest(r *http.Request) (primitive.ObjectID, error) {
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

func (h *RoomHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

func (h *RoomHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

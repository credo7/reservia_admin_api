// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"fmt"
	"reservia-admin-api/internal/middleware"
	"reservia-admin-api/internal/model"
	"reservia-admin-api/internal/service"
	"net/http"
	"strings"
	"time"

	chi "github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"reservia-admin-api/pkg/logger"
	"reservia-admin-api/pkg/validator"
)

// RestaurantHandler handles restaurant-related HTTP requests.
type RestaurantHandler struct {
	restaurantService             *service.RestaurantService
	reservationService            *service.ReservationService
	restaurantAvailabilityService *service.RestaurantAvailabilityService
	authService                   *service.AuthService
	validator                     *validator.Validator
	logger                        logger.Logger
}

// NewRestaurantHandler creates a new restaurant handler.
func NewRestaurantHandler(restaurantService *service.RestaurantService, reservationService *service.ReservationService, restaurantAvailabilityService *service.RestaurantAvailabilityService, authService *service.AuthService, logger logger.Logger) *RestaurantHandler {
	return &RestaurantHandler{
		restaurantService:             restaurantService,
		reservationService:            reservationService,
		restaurantAvailabilityService: restaurantAvailabilityService,
		authService:                   authService,
		validator:                     validator.New(),
		logger:                        logger,
	}
}

// CreateRestaurant handles POST /restaurants.
//
//	@Summary		Create a new restaurant (bootstrap)
//	@Description	Create a new restaurant with the provided information. The creating employee automatically becomes the owner.
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurant	body		model.CreateRestaurantRequest	true	"Restaurant creation data"
//	@Success		201			{object}	model.RestaurantSummary		"Restaurant created successfully"
//	@Failure		400			{object}	map[string]string					"Invalid request body"
//	@Failure		401			{object}	map[string]string					"Authentication required"
//	@Failure		500			{object}	map[string]string					"Internal server error"
//	@Router			/restaurants [post]
//	@Security		BearerAuth
func (h *RestaurantHandler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	creator, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during CreateRestaurant request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req model.CreateRestaurantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request data using struct tags
	if err := h.validator.Struct(req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create restaurant with bootstrap logic (creator becomes owner)
	createdRestaurant, err := h.restaurantService.CreateRestaurant(r.Context(), creator, &req)
	if err != nil {
		h.logger.Error("Failed to create restaurant", "error", err, "creator_id", creator.ID)
		if strings.Contains(err.Error(), "ownership") {
			h.writeError(w, http.StatusInternalServerError, "Failed to establish restaurant ownership")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to create restaurant")
		}
		return
	}

	h.logger.Info("Restaurant created successfully by employee",
		"restaurant_id", createdRestaurant.ID.Hex(),
		"creator_id", creator.ID.Hex(),
		"restaurant_name", createdRestaurant.Name)
	h.writeJSON(w, http.StatusCreated, createdRestaurant.ToSummary())
}

// GetRestaurant handles GET /restaurants/{id}.
//
//	@Summary		Get restaurant by ID
//	@Description	Retrieve a restaurant by its unique ID
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string						true	"Restaurant ID"
//	@Success		200	{object}	model.Restaurant	"Restaurant found"
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

	rest, err := h.restaurantService.GetRestaurantByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get restaurant", "error", err)
		h.writeError(w, http.StatusNotFound, "Restaurant not found")
		return
	}

	h.writeJSON(w, http.StatusOK, rest)
}

// UpdateRestaurant handles PATCH /restaurants/{id}.
//
//	@Summary		Update restaurant information
//	@Description	Update restaurant details. City cannot be changed after creation.
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string							true	"Restaurant ID"
//	@Param			restaurant	body		model.UpdateRestaurantRequest	true	"Restaurant update data"
//	@Success		200			{object}	model.Restaurant				"Restaurant updated successfully"
//	@Failure		400			{object}	map[string]string				"Invalid request"
//	@Failure		401			{object}	map[string]string				"Authentication required"
//	@Failure		404			{object}	map[string]string				"Restaurant not found"
//	@Failure		500			{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{id} [patch]
//	@Security		BearerAuth
func (h *RestaurantHandler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.UpdateRestaurantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedRestaurant, err := h.restaurantService.UpdateRestaurant(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update restaurant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update restaurant")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedRestaurant)
}

// DeleteRestaurant handles DELETE /restaurants/{id}.
//
//	@Summary		Delete a restaurant
//	@Description	Delete a restaurant by ID
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Restaurant ID"
//	@Success		204			"Restaurant deleted successfully"
//	@Failure		400	{object}	map[string]string	"Invalid restaurant ID"
//	@Failure		401	{object}	map[string]string	"Authentication required"
//	@Failure		404	{object}	map[string]string	"Restaurant not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{id} [delete]
//	@Security		BearerAuth
func (h *RestaurantHandler) DeleteRestaurant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	if err := h.restaurantService.DeleteRestaurant(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete restaurant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to delete restaurant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListRestaurants handles GET /restaurants.
//
//	@Summary		List associated restaurants
//	@Description	Retrieve all restaurants associated with the authenticated employee
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Success		200			{array}		model.RestaurantSummary	"List of associated restaurants"
//	@Failure		401			{object}	map[string]string			"Unauthorized - authentication required"
//	@Failure		500			{object}	map[string]string			"Internal server error"
//	@Router			/restaurants [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) ListRestaurants(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during ListRestaurants")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get employee's restaurant IDs
	restaurantIDs := employee.GetRestaurantIDs()
	if len(restaurantIDs) == 0 {
		// Return empty array if employee has no restaurants
		h.writeJSON(w, http.StatusOK, []*model.RestaurantSummary{})
		return
	}

	// Get all restaurants associated with this employee
	restaurants, err := h.restaurantService.GetRestaurantsByIDs(r.Context(), restaurantIDs)
	if err != nil {
		h.logger.Error("Failed to get associated restaurants", "error", err, "employee_id", employee.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to list restaurants")
		return
	}

	// Convert to summary format (matching Python RestaurantMiniSchema)
	summaries := make([]*model.RestaurantSummary, len(restaurants))
	for i, rest := range restaurants {
		summaries[i] = rest.ToSummary()
	}

	h.logger.Info("Listed associated restaurants", "employee_id", employee.ID, "count", len(summaries))
	h.writeJSON(w, http.StatusOK, summaries)
}

// GetRestaurantRooms handles GET /restaurants/{id}/rooms.
//
//	@Summary		Get restaurant rooms
//	@Description	Retrieve all rooms for a specific restaurant
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restaurant ID"
//	@Success		200	{object}	map[string]interface{}	"Restaurant rooms with count"
//	@Failure		400	{object}	map[string]string		"Invalid restaurant ID"
//	@Failure		404	{object}	map[string]string		"Restaurant not found"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{id}/rooms [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) GetRestaurantRooms(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	rooms, err := h.restaurantService.GetRestaurantRooms(r.Context(), id)
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
//
//	@Summary		Get a specific room from a restaurant
//	@Description	Retrieve a specific room by ID from a restaurant
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Restaurant ID"
//	@Param			roomId	path		string			true	"Room ID"
//	@Success		200		{object}	model.Room		"Room found successfully"
//	@Failure		400		{object}	map[string]string	"Invalid restaurant ID or room ID"
//	@Failure		404		{object}	map[string]string	"Room not found"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/restaurants/{id}/rooms/{roomId} [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) GetRestaurantRoom(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	roomIDStr := chi.URLParam(r, "roomId")
	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid room ID")
		return
	}

	room, err := h.restaurantService.GetRestaurantRoom(r.Context(), id, roomID)
	if err != nil {
		h.logger.Error("Failed to get restaurant room", "error", err)
		h.writeError(w, http.StatusNotFound, "Room not found")
		return
	}

	h.writeJSON(w, http.StatusOK, room)
}

// CheckURLNameAvailability handles GET /restaurants/url-name-availability.
//
//	@Summary		Check URL name availability
//	@Description	Check if a URL name is available for restaurant creation
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			url_name	query		string							true	"URL name to check"
//	@Success		200			{object}	model.UrlNameAvailabilityResponse	"URL name availability"
//	@Failure		400			{object}	map[string]string				"Missing url_name parameter"
//	@Failure		500			{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/url-name-availability [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) CheckURLNameAvailability(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("URL name availability check requested", "employee_id", employeeID)

	urlName := r.URL.Query().Get("url_name")
	if urlName == "" {
		h.writeError(w, http.StatusBadRequest, "url_name parameter is required")
		return
	}

	availability, err := h.restaurantService.CheckURLNameAvailability(r.Context(), urlName)
	if err != nil {
		h.logger.Error("Failed to check URL name availability", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to check URL name availability")
		return
	}

	h.writeJSON(w, http.StatusOK, availability)
}

// GetRestaurantByURLNameOrID handles GET /restaurants/{urlNameOrRestId}.
//
//	@Summary		Get restaurant by URL name or ID
//	@Description	Retrieve a restaurant by its URL name or ObjectID
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			urlNameOrRestId	path		string						true	"Restaurant URL name or ID"
//	@Success		200				{object}	model.Restaurant	"Restaurant found"
//	@Failure		404				{object}	map[string]string		"Restaurant not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{urlNameOrRestId} [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) GetRestaurantByURLNameOrID(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Restaurant retrieval requested", "employee_id", employeeID)

	urlNameOrID := chi.URLParam(r, "urlNameOrRestId")
	if urlNameOrID == "" {
		h.writeError(w, http.StatusBadRequest, "URL name or restaurant ID is required")
		return
	}

	restaurant, err := h.restaurantService.GetRestaurantByURLNameOrID(r.Context(), urlNameOrID)
	if err != nil {
		h.logger.Error("Failed to get restaurant", "url_name_or_id", urlNameOrID, "error", err)
		h.writeError(w, http.StatusNotFound, "Restaurant not found")
		return
	}

	h.writeJSON(w, http.StatusOK, restaurant)
}

// UpdateRestaurantSettings handles PATCH /restaurants/{restaurantId}/settings.
//
//	@Summary		Update restaurant settings
//	@Description	Update restaurant settings only
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string								true	"Restaurant ID"
//	@Param			settings		body		model.UpdateRestaurantSettingsRequest	true	"Restaurant settings update data"
//	@Success		200				{object}	model.Restaurant				"Restaurant settings updated successfully"
//	@Failure		400				{object}	map[string]string					"Invalid request"
//	@Failure		404				{object}	map[string]string					"Restaurant not found"
//	@Failure		500				{object}	map[string]string					"Internal server error"
//	@Router			/restaurants/{restaurantId}/settings [patch]
//	@Security		BearerAuth
func (h *RestaurantHandler) UpdateRestaurantSettings(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Restaurant settings update requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.UpdateRestaurantSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedRestaurant, err := h.restaurantService.UpdateRestaurantSettings(r.Context(), restaurantID, &req)
	if err != nil {
		h.logger.Error("Failed to update restaurant settings", "restaurant_id", restaurantID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update restaurant settings")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedRestaurant)
}

// EnableRestaurant handles POST /restaurants/{restaurantId}/enable.
//
//	@Summary		Enable restaurant
//	@Description	Enable a restaurant (set isActive to true)
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Success		200				{object}	model.Restaurant	"Restaurant enabled successfully"
//	@Failure		400				{object}	map[string]string		"Invalid restaurant ID"
//	@Failure		404				{object}	map[string]string		"Restaurant not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/enable [post]
//	@Security		BearerAuth
func (h *RestaurantHandler) EnableRestaurant(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Restaurant enable requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	enabledRestaurant, err := h.restaurantService.EnableRestaurant(r.Context(), restaurantID)
	if err != nil {
		h.logger.Error("Failed to enable restaurant", "restaurant_id", restaurantID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to enable restaurant")
		return
	}

	h.writeJSON(w, http.StatusOK, enabledRestaurant)
}

// DisableRestaurant handles POST /restaurants/{restaurantId}/disable.
//
//	@Summary		Disable restaurant
//	@Description	Disable a restaurant (set isActive to false)
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string					true	"Restaurant ID"
//	@Success		200				{object}	model.Restaurant	"Restaurant disabled successfully"
//	@Failure		400				{object}	map[string]string		"Invalid restaurant ID"
//	@Failure		404				{object}	map[string]string		"Restaurant not found"
//	@Failure		500				{object}	map[string]string		"Internal server error"
//	@Router			/restaurants/{restaurantId}/disable [post]
//	@Security		BearerAuth
func (h *RestaurantHandler) DisableRestaurant(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Restaurant disable requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	disabledRestaurant, err := h.restaurantService.DisableRestaurant(r.Context(), restaurantID)
	if err != nil {
		h.logger.Error("Failed to disable restaurant", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Restaurant not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to disable restaurant")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, disabledRestaurant)
}

// GetRestaurantAvailability handles GET /restaurants/{urlNameOrRestId}/availability.
//
//	@Summary		Get restaurant availability
//	@Description	Get restaurant availability for a specific date
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			urlNameOrRestId	path		string									true	"Restaurant URL name or ID"
//	@Param			date			query		string									false	"Date in YYYY-MM-DD format (default: today)"
//	@Success		200				{object}	model.RestaurantAvailability	"Restaurant availability"
//	@Failure		400				{object}	map[string]string						"Invalid date format"
//	@Failure		404				{object}	map[string]string						"Restaurant not found"
//	@Failure		500				{object}	map[string]string						"Internal server error"
//	@Router			/restaurants/{urlNameOrRestId}/availability [get]
func (h *RestaurantHandler) GetRestaurantAvailability(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint does not require authentication (public availability check)
	urlNameOrID := chi.URLParam(r, "urlNameOrRestId")
	if urlNameOrID == "" {
		h.writeError(w, http.StatusBadRequest, "URL name or restaurant ID is required")
		return
	}

	dateStr := r.URL.Query().Get("date")
	var chosenDate time.Time
	var err error

	if dateStr == "" {
		// Default to today
		chosenDate = time.Now().UTC()
	} else {
		chosenDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
	}

	availability, err := h.restaurantAvailabilityService.GetAvailability(r.Context(), urlNameOrID, chosenDate)
	if err != nil {
		h.logger.Error("Failed to get restaurant availability", "url_name_or_id", urlNameOrID, "date", chosenDate.Format("2006-01-02"), "error", err)
		h.writeError(w, http.StatusNotFound, "Restaurant not found or invalid date")
		return
	}

	h.writeJSON(w, http.StatusOK, availability)
}

// CreateSubURL handles POST /restaurants/{restaurantId}/sub-url.
//
//	@Summary		Create a new sub-URL for a restaurant
//	@Description	Create a new sub-URL for tracking reservation sources
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string							true	"Restaurant ID"
//	@Param			subURL			body		model.CreateSubURLRequest		true	"Sub-URL creation data"
//	@Success		200				{object}	model.Restaurant		"Sub-URL created successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Restaurant not found"
//	@Failure		409				{object}	map[string]string				"Sub-URL name or key already exists"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{restaurantId}/sub-url [post]
//	@Security		BearerAuth
func (h *RestaurantHandler) CreateSubURL(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Sub-URL creation requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.CreateSubURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedRestaurant, err := h.restaurantService.AddSubURL(r.Context(), restaurantID, &req)
	if err != nil {
		h.logger.Error("Failed to create sub-URL", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Restaurant not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to create sub-URL")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, updatedRestaurant)
}

// DeleteSubURL handles DELETE /restaurants/{restaurantId}/sub-url/{subUrlId}.
//
//	@Summary		Delete a sub-URL from a restaurant
//	@Description	Remove a sub-URL from a restaurant
//	@Tags			restaurants
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string							true	"Restaurant ID"
//	@Param			subUrlId		path		string							true	"Sub-URL ID"
//	@Success		200				{object}	model.Restaurant		"Sub-URL deleted successfully"
//	@Failure		400				{object}	map[string]string				"Invalid request"
//	@Failure		404				{object}	map[string]string				"Restaurant or sub-URL not found"
//	@Failure		500				{object}	map[string]string				"Internal server error"
//	@Router			/restaurants/{restaurantId}/sub-url/{subUrlId} [delete]
//	@Security		BearerAuth
func (h *RestaurantHandler) DeleteSubURL(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Sub-URL deletion requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	subURLIDStr := chi.URLParam(r, "subUrlId")
	if subURLIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "Sub-URL ID is required")
		return
	}

	subURLID, err := primitive.ObjectIDFromHex(subURLIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid sub-URL ID")
		return
	}

	updatedRestaurant, err := h.restaurantService.DeleteSubURL(r.Context(), restaurantID, subURLID)
	if err != nil {
		h.logger.Error("Failed to delete sub-URL", "restaurant_id", restaurantID, "sub_url_id", subURLID.Hex(), "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to delete sub-URL")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, updatedRestaurant)
}

// MarkReservationsAsSeen handles PATCH /restaurants/{restaurantId}/reservations/mark_seen.
//
//	@Summary		Mark reservations as seen
//	@Description	Mark specific reservations or all unseen reservations as seen for a restaurant
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string								true	"Restaurant ID"
//	@Param			markSeenData	body		model.MarkReservationsAsSeenRequest	true	"Mark seen request data"
//	@Success		200				{object}	map[string]string						"Reservations marked as seen successfully"
//	@Failure		400				{object}	map[string]string						"Invalid request"
//	@Failure		404				{object}	map[string]string						"Restaurant not found"
//	@Failure		500				{object}	map[string]string						"Internal server error"
//	@Router			/restaurants/{restaurantId}/reservations/mark_seen [patch]
//	@Security		BearerAuth
func (h *RestaurantHandler) MarkReservationsAsSeen(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Mark reservations as seen requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	var req model.MarkReservationsAsSeenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.reservationService.MarkReservationsAsSeen(r.Context(), restaurantID, &req, employeeID); err != nil {
		h.logger.Error("Failed to mark reservations as seen", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to mark reservations as seen")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Reservations marked as seen successfully",
	})
}

// GetReservationCounts handles GET /restaurants/{restaurantId}/reservations/counts.
//
//	@Summary		Get reservation counts
//	@Description	Get counts of unseen and pending reservations for a restaurant
//	@Tags			reservations
//	@Accept			json
//	@Produce		json
//	@Param			restaurantId	path		string								true	"Restaurant ID"
//	@Success		200				{object}	model.ReservationCountsResponse	"Reservation counts retrieved successfully"
//	@Failure		400				{object}	map[string]string					"Invalid request"
//	@Failure		404				{object}	map[string]string					"Restaurant not found"
//	@Failure		500				{object}	map[string]string					"Internal server error"
//	@Router			/restaurants/{restaurantId}/reservations/counts [get]
//	@Security		BearerAuth
func (h *RestaurantHandler) GetReservationCounts(w http.ResponseWriter, r *http.Request) {
	// Authentication required - extract employee ID from token
	employeeID, err := h.authenticateRequest(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.logger.Info("Reservation counts requested", "employee_id", employeeID)

	restaurantIDStr := chi.URLParam(r, "restaurantId")
	restaurantID, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid restaurant ID")
		return
	}

	counts, err := h.reservationService.GetReservationCounts(r.Context(), restaurantID)
	if err != nil {
		h.logger.Error("Failed to get reservation counts", "restaurant_id", restaurantID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Restaurant not found")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to get reservation counts")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, counts)
}

// Helper methods

// authenticateRequest extracts and validates the Bearer token, returning the employee ID.
func (h *RestaurantHandler) authenticateRequest(r *http.Request) (primitive.ObjectID, error) {
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

func (h *RestaurantHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

func (h *RestaurantHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

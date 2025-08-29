package handler

import (
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"reservia-admin-api/internal/model"
	"reservia-admin-api/internal/service"
	"reservia-admin-api/pkg/logger"
)

type CityHandler struct {
	cityService *service.CityService
	logger      logger.Logger
}

func NewCityHandler(cityService *service.CityService, logger logger.Logger) *CityHandler {
	return &CityHandler{
		cityService: cityService,
		logger:      logger,
	}
}

// GetCities godoc
// @Summary      Get all cities
// @Description  Retrieve all available cities with their information
// @Tags         cities
// @Accept       json
// @Produce      json
// @Success      200  {array}   model.CityResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/cities [get]
func (h *CityHandler) GetCities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var cities []*model.City
	var err error

	cities, err = h.cityService.GetAllCities(ctx)

	if err != nil {
		http.Error(w, "Failed to retrieve cities", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []*model.CityResponse
	for _, c := range cities {
		response = append(response, c.ToResponse())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode cities response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// GetCityByName godoc
// @Summary      Get city by name
// @Description  Retrieve a specific city by its name
// @Tags         cities
// @Accept       json
// @Produce      json
// @Param        name   path      string  true  "City name"
// @Success      200  {object}  model.CityResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/cities/{name} [get]
func (h *CityHandler) GetCityByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract city name from URL parameter
	name := chi.URLParam(r, "name")

	cityEntity, err := h.cityService.GetCityByName(ctx, name)
	if err != nil {
		http.Error(w, "City not found", http.StatusNotFound)
		return
	}

	response := cityEntity.ToResponse()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode city response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

package handler

import (
	"encoding/json"
	"net/http"

	"github.com/reservia/api/internal/domain/city"
	cityUseCase "github.com/reservia/api/internal/usecase/city"
)

type CityHandler struct {
	cityUseCase *cityUseCase.UseCase
}

func NewCityHandler(cityUseCase *cityUseCase.UseCase) *CityHandler {
	return &CityHandler{
		cityUseCase: cityUseCase,
	}
}

// GetCities godoc
// @Summary      Get all cities
// @Description  Retrieve all available cities with their information
// @Tags         cities
// @Accept       json
// @Produce      json
// @Param        country  query      string  false  "Filter by country code or name"
// @Success      200  {array}   city.CityResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/cities [get]
func (h *CityHandler) GetCities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if country filter is provided
	country := r.URL.Query().Get("country")

	var cities []*city.City
	var err error

	if country != "" {
		cities, err = h.cityUseCase.GetCitiesByCountry(ctx, country)
	} else {
		cities, err = h.cityUseCase.GetAllCities(ctx)
	}

	if err != nil {
		http.Error(w, "Failed to retrieve cities", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []*city.CityResponse
	for _, c := range cities {
		response = append(response, c.ToResponse())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetCityByName godoc
// @Summary      Get city by name
// @Description  Retrieve a specific city by its name
// @Tags         cities
// @Accept       json
// @Produce      json
// @Param        name   path      string  true  "City name"
// @Success      200  {object}  city.CityResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/cities/{name} [get]
func (h *CityHandler) GetCityByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract city name from URL path
	name := r.URL.Path[len("/api/v1/cities/"):]

	cityEntity, err := h.cityUseCase.GetCityByName(ctx, name)
	if err != nil {
		http.Error(w, "City not found", http.StatusNotFound)
		return
	}

	response := cityEntity.ToResponse()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

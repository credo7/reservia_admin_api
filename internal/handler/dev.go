package handler

import (
	"net/http"
)

type DevHandler struct{}

func NewDevHandler() *DevHandler {
	return &DevHandler{}
}

// Ping godoc
// @Summary      Development ping endpoint
// @Description  Simple ping endpoint for development and health checking
// @Tags         development
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /api/v1/dev/ping [get]
func (h *DevHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "OK!"}`))
}

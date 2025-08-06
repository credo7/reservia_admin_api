package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// GetMetrics godoc
// @Summary      Prometheus metrics endpoint
// @Description  Returns metrics in Prometheus format for scraping
// @Tags         metrics
// @Produce      text/plain
// @Success      200  {string}  string
// @Router       /metrics [get]
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// Set headers for Prometheus
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Use Prometheus handler
	promhttp.Handler().ServeHTTP(w, r)
}

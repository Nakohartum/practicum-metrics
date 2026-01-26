package handler

import (
	"net/http"

	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

type MetricsHandler struct {
	service *service.MetricsService
}

func NewMetricsHandler(s *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: s,
	}
}

func (mh *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")
	

	if metricName == "" {
		http.Error(w, "no metric's name", http.StatusNotFound)
		return
	}

	if err := mh.service.SetData(metricType, metricName, metricValue); err != nil {
		http.Error(w, "error setting metric data", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}	
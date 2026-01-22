package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Nakohartum/practicum-metrics/internal/service"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")

	parts := strings.Split(path, "/")

	if len(parts) < 3 || parts[1] == "" {
		http.Error(w, "No metric's name", http.StatusNotFound)
		return
	}

	metricType := parts[0]
	metricKey := parts[1]
	metricValue := parts[2]

	
	if err := mh.service.SetData(metricType, metricKey, metricValue); err != nil {
		http.Error(w, "Error setting metric data", http.StatusBadRequest)
		return
	}

	res, err := mh.service.GetData(path)

	if err == nil {
		fmt.Println(res)
	} else{
		fmt.Println(err)
	}


	w.WriteHeader(http.StatusOK)
}
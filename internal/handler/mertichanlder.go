package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func readBody(r *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeJSONBody[T any](r *http.Request, dst *T) error {
	body, err := readBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

type MetricsHandler struct {
	service service.Service
}

type metricValidationError struct {
	message string
}

func (e metricValidationError) Error() string {
	return e.message
}

func NewMetricsHandler(s service.Service) *MetricsHandler {
	return &MetricsHandler{
		service: s,
	}
}

func (mh *MetricsHandler) checkError(err error) config.PGErrorClassification {
	validator := config.NewPostgresErrorClassifier()
	return validator.Classify(err)
}

func (mh *MetricsHandler) UpdateMetricsDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		var metric models.Metrics
		if err := decodeJSONBody(r, &metric); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if metric.ID == "" {
			http.Error(w, "no metric's name", http.StatusBadRequest)
			return
		}
		if metric.MType == "" {
			http.Error(w, "no metric's type", http.StatusBadRequest)
			return
		}
		if err := mh.saveMetric(metric, true); err != nil {
			var validationErr metricValidationError
			if errors.As(err, &validationErr) {
				http.Error(w, validationErr.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fun)
}

func (mh *MetricsHandler) GetMetricsByNameHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		var metricToSearch models.Metrics
		if err := decodeJSONBody(r, &metricToSearch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if metricToSearch.ID == "" || metricToSearch.MType == "" {
			http.Error(w, "metric's name and type are required", http.StatusBadRequest)
			return
		}
		metricData, err := mh.service.GetData(metricToSearch.MType, metricToSearch.ID)
		if err != nil {
			http.Error(w, "no metric found", http.StatusNotFound)
			return
		}

		responseData, err := json.Marshal(metricData)
		if err != nil {
			http.Error(w, "error marshaling response data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseData)

	}
	return http.HandlerFunc(fun)
}

func (mh *MetricsHandler) SetMetricDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")
		metricValue := chi.URLParam(r, "metricValue")

		if metricName == "" {
			http.Error(w, "no metric's name", http.StatusNotFound)
			return
		}

		if err := mh.setDataWithRetry(metricType, metricName, metricValue); err != nil {
			http.Error(w, "error setting metric data", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fun)
}


func (mh *MetricsHandler) GetMetricDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}

		metricType := chi.URLParam(r, "metricType")
		metricName := chi.URLParam(r, "metricName")

		if metricName == "" || metricType == "" {
			http.Error(w, "no metric found", http.StatusNotFound)
			return
		}

		metricData, err := mh.service.GetData(metricType, metricName)

		if err != nil {
			http.Error(w, "no metric found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

		switch metricType {
		case models.Counter:
			w.Write([]byte(strconv.FormatInt(*metricData.Delta, 10)))
		case models.Gauge:
			w.Write([]byte(strconv.FormatFloat(*metricData.Value, 'f', -1, 64)))
		}

	}
	return http.HandlerFunc(fun)
}

type PageHandler struct {
	service repository.Storage
	tpl     *template.Template
}

func NewPageHandler(s repository.Storage) *PageHandler {
	const page = `
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Metrics</title>
  <style>
    body { font-family: sans-serif; padding: 24px; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: 8px; }
    th { text-align: left; }
  </style>
</head>
<body>
  <h1>Metrics</h1>
  {{if .}}
  <table>
    <thead>
      <tr><th>Type</th><th>Name</th><th>Value</th></tr>
    </thead>
    <tbody>
      {{range .}}
        <tr>
          <td>{{.Type}}</td>
          <td>{{.Name}}</td>
          <td>{{.Value}}</td>
        </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
    <p>Метрик пока нет.</p>
  {{end}}
</body>
</html>`
	return &PageHandler{
		service: s,
		tpl:     template.Must(template.New("page").Parse(page)),
	}
}

func (mh *MetricsHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := mh.service.GetAll()

	w.WriteHeader(http.StatusOK)

	if err := NewPageHandler(mh.service).tpl.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

func (mh *MetricsHandler) Ping(ctx context.Context) http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}

		err := mh.service.Ping(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fun)
}

func (mh *MetricsHandler) SetMetricsDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		var metrics []models.Metrics
		body, err := readBody(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Accept both batch array and single metric payload.
		if err = json.Unmarshal(body, &metrics); err != nil {
			var single models.Metrics
			if errSingle := json.Unmarshal(body, &single); errSingle != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			metrics = []models.Metrics{single}
		}
		counterBatch := make(map[string]int64)
		gaugeBatch := make(map[string]float64)

		for _, metric := range metrics {
			if metric.ID == "" {
				continue
			}
			switch metric.MType {
			case models.Counter:
				if metric.Delta == nil {
					continue
				}
				counterBatch[metric.ID] += *metric.Delta
			case models.Gauge:
				if metric.Value == nil {
					continue
				}
				// In one batch, the latest gauge value for the same metric wins.
				gaugeBatch[metric.ID] = *metric.Value
			default:
				continue
			}
		}

		for id, delta := range counterBatch {
			_ = mh.setDataWithRetry(models.Counter, id, strconv.FormatInt(delta, 10))
		}

		for id, value := range gaugeBatch {
			_ = mh.setDataWithRetry(models.Gauge, id, strconv.FormatFloat(value, 'f', -1, 64))
		}
		w.WriteHeader(http.StatusOK)
	}

	return http.HandlerFunc(fun)
}

func (mh *MetricsHandler) setDataWithRetry(metricType, metricName, metricValue string) error {
	err := mh.service.SetData(metricType, metricName, metricValue)
	if err == nil {
		return nil
	}
	if mh.checkError(err) != config.Retriable {
		return err
	}

	retries := 3
	cooldown := 1 * time.Second
	for i := 0; i < retries; i++ {
		err = mh.service.SetData(metricType, metricName, metricValue)
		if err == nil {
			return nil
		}
		time.Sleep(cooldown)
		cooldown += 2
	}
	return err
}

func (mh *MetricsHandler) saveMetric(metric models.Metrics, strict bool) error {
	value, skip, err := getMetricValue(metric, strict)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	return mh.service.SetData(metric.MType, metric.ID, value)
}

func getMetricValue(metric models.Metrics, strict bool) (string, bool, error) {
	switch metric.MType {
	case models.Counter:
		if metric.Delta == nil {
			if strict {
				return "", false, metricValidationError{message: "delta value is required for counter type"}
			}
			return "", true, nil
		}
		return strconv.FormatInt(*metric.Delta, 10), false, nil
	case models.Gauge:
		if metric.Value == nil {
			if strict {
				return "", false, metricValidationError{message: "value is required for gauge type"}
			}
			return "", true, nil
		}
		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), false, nil
	default:
		if strict {
			return "", false, metricValidationError{message: "unsupported metric type"}
		}
		return "", true, nil
	}
}

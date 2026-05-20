package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Nakohartum/practicum-metrics/internal/audit"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/service"
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
	return json.NewDecoder(r.Body).Decode(dst)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

// MetricsHandler exposes HTTP handlers for metric operations.
type MetricsHandler struct {
	service service.Service
	auditor *audit.Auditor
}

// NewMetricsHandler creates a MetricsHandler with an optional auditor.
func NewMetricsHandler(s service.Service, auditors ...*audit.Auditor) *MetricsHandler {
	var auditor *audit.Auditor
	if len(auditors) > 0 {
		auditor = auditors[0]
	}

	return &MetricsHandler{
		service: s,
		auditor: auditor,
	}
}

// UpdateMetricsDataHandle returns a handler for updating one JSON metric.
func (mh *MetricsHandler) UpdateMetricsDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		var metric models.Metrics
		if err := decodeJSONBody(r, &metric); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if metric.ID == "" {
			writeJSONError(w, "no metric's name", http.StatusBadRequest)
			return
		}
		if metric.MType == "" {
			writeJSONError(w, "no metric's type", http.StatusBadRequest)
			return
		}
		if err := mh.service.SetDataUsingMetrics(r.Context(), []models.Metrics{metric}); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		mh.notifyAudit(r, []string{metric.ID})
	}
	return http.HandlerFunc(fun)
}

// GetMetricsByNameHandle returns a handler for reading one JSON metric.
func (mh *MetricsHandler) GetMetricsByNameHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		var metricToSearch models.Metrics
		if err := decodeJSONBody(r, &metricToSearch); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if metricToSearch.ID == "" || metricToSearch.MType == "" {
			writeJSONError(w, "metric's name and type are required", http.StatusBadRequest)
			return
		}
		metricData, err := mh.service.GetData(r.Context(), metricToSearch.MType, metricToSearch.ID)
		if err != nil {
			writeJSONError(w, "no metric found", http.StatusNotFound)
			return
		}

		responseData, err := json.Marshal(metricData)
		if err != nil {
			writeJSONError(w, "error marshaling response data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseData)
	}
	return http.HandlerFunc(fun)
}

// SetMetricDataHandle returns a handler for updating one metric from URL params.
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

		if err := mh.service.SetData(r.Context(), metricType, metricName, metricValue); err != nil {
			http.Error(w, "error setting metric data", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		mh.notifyAudit(r, []string{metricName})
	}
	return http.HandlerFunc(fun)
}

// GetMetricDataHandle returns a handler for reading one metric from URL params.
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

		metricData, err := mh.service.GetData(r.Context(), metricType, metricName)

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

// PageHandler renders an HTML page with stored metrics.
type PageHandler struct {
	service service.Service
	tpl     *template.Template
}

// NewPageHandler creates a PageHandler for the provided storage.
func NewPageHandler(s service.Service) *PageHandler {
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
          <td>{{.MType}}</td>
          <td>{{.ID}}</td>
          <td>{{if .Delta}}{{.Delta}}{{else}}{{.Value}}{{end}}</td>
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

// ServePage writes the metrics overview HTML page.
func (mh *MetricsHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := mh.service.GetAll(r.Context())

	w.WriteHeader(http.StatusOK)

	if err := NewPageHandler(mh.service).tpl.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

// Ping returns a handler for storage health checks.
func (mh *MetricsHandler) Ping() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}

		err := mh.service.Ping(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fun)
}

// SetMetricsDataHandle returns a handler for updating a batch of JSON metrics.
func (mh *MetricsHandler) SetMetricsDataHandle() http.Handler {
	fun := func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		metrics, err := decodeMetricsPayload(r)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		metricsNames := make([]string, 0, len(metrics))
		seenMetricNames := make(map[string]struct{})

		for _, metric := range metrics {
			if metric.ID == "" {
				continue
			}
			if _, ok := seenMetricNames[metric.ID]; ok {
				continue
			}
			seenMetricNames[metric.ID] = struct{}{}
			metricsNames = append(metricsNames, metric.ID)
		}

		if err := mh.service.SetDataUsingMetrics(r.Context(), metrics); err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		mh.notifyAudit(r, metricsNames)
	}

	return http.HandlerFunc(fun)
}

func decodeMetricsPayload(r *http.Request) ([]models.Metrics, error) {
	br := bufio.NewReader(r.Body)
	first, err := firstNonSpaceByte(br)
	if err != nil {
		return nil, err
	}
	if err := br.UnreadByte(); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(br)
	if first == '[' {
		var metrics []models.Metrics
		if err := decoder.Decode(&metrics); err != nil {
			return nil, err
		}
		return metrics, nil
	}

	var metric models.Metrics
	if err := decoder.Decode(&metric); err != nil {
		return nil, err
	}
	return []models.Metrics{metric}, nil
}

func firstNonSpaceByte(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return b, nil
		}
	}
}

func (mh *MetricsHandler) notifyAudit(r *http.Request, metrics []string) {
	if mh.auditor == nil || !mh.auditor.Enabled() || len(metrics) == 0 {
		return
	}

	mh.auditor.Notify(r.Context(), audit.Event{
		Ts:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

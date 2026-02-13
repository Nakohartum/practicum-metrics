package handler

import (
	"html/template"
	"net/http"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
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

func (mh *MetricsHandler) SetMetricDataHandle() http.Handler {
	fun := func (w http.ResponseWriter, r *http.Request)  {
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
	return http.HandlerFunc(fun)
}	


func (mh *MetricsHandler) GetMetricDataHandle(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricName == "" || metricType == ""{
		http.Error(w, "no metric found", http.StatusNotFound)
	}

	metricData, err := mh.service.GetData(metricType, metricName)

	if err != nil{
		http.Error(w, "no metric found", http.StatusNotFound)
		return
	}
 
	w.Write([]byte(metricData.Value))
	w.WriteHeader(http.StatusOK)
}

type PageHandler struct {
	service repository.Storage
	tpl *template.Template
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

func (mh *MetricsHandler) ServePage(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	data := mh.service.GetAll()

	w.WriteHeader(http.StatusOK)

	if err := NewPageHandler(mh.service).tpl.Execute(w, data); err != nil{
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	
}
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func withRouteParams(req *http.Request, params map[string]string) *http.Request {
	routeCtx := chi.NewRouteContext()
	for key, value := range params {
		routeCtx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestRequireMethod(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		required   string
		wantResult bool
		wantStatus int
	}{
		{name: "allows matching method", method: http.MethodGet, required: http.MethodGet, wantResult: true, wantStatus: http.StatusOK},
		{name: "rejects different method", method: http.MethodPost, required: http.MethodGet, wantResult: false, wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			rr := httptest.NewRecorder()

			got := requireMethod(rr, req, tt.required)

			assert.Equal(t, tt.wantResult, got)
			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestReadBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "reads request body", body: `{"id":"metric"}`},
		{name: "reads empty body", body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))

			got, err := readBody(req)

			require.NoError(t, err)
			assert.Equal(t, tt.body, string(got))
		})
	}
}

func TestDecodeJSONBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    models.Metrics
		wantErr bool
	}{
		{
			name: "decodes valid json",
			body: `{"id":"hits","type":"counter","delta":3}`,
			want: models.Metrics{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
		},
		{
			name:    "returns error for invalid json",
			body:    `{"id":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var got models.Metrics

			err := decodeJSONBody(req, &got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewMetricsHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates handler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			handler := NewMetricsHandler(svc)

			require.NotNil(t, handler)
			assert.Equal(t, svc, handler.service)
		})
	}
}

func TestMetricsHandlerCheckError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantCls config.PGErrorClassification
	}{
		{name: "nil is non retriable", err: nil, wantCls: config.NonRetriable},
		{name: "connection failure is retriable", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, wantCls: config.Retriable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &MetricsHandler{}
			assert.Equal(t, tt.wantCls, handler.checkError(tt.err))
		})
	}
}

func TestMetricsHandlerUpdateMetricsDataHandle(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mock       func(*mocks.MockService)
		wantStatus int
	}{
		{
			name:       "returns bad request for invalid json",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request for missing id",
			body:       `{"type":"counter","delta":3}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request for missing type",
			body:       `{"id":"hits","delta":3}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request for missing delta in strict mode",
			body:       `{"id":"hits","type":"counter"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "saves valid metric",
			body: `{"id":"hits","type":"counter","delta":3}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "3").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "returns internal error for service failure",
			body: `{"id":"hits","type":"counter","delta":3}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "3").Return(assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			handler.UpdateMetricsDataHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMetricsHandlerGetMetricsByNameHandle(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		mock         func(*mocks.MockService)
		wantStatus   int
		wantBodyPart string
	}{
		{
			name:       "returns bad request for invalid json",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request for missing fields",
			body:       `{"id":"hits"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "returns not found when service fails",
			body: `{"id":"hits","type":"counter"}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{}, assert.AnError)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "returns metric json",
			body: `{"id":"hits","type":"counter"}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{
					ID:    "hits",
					MType: models.Counter,
					Delta: ptrInt64(5),
				}, nil)
			},
			wantStatus:   http.StatusOK,
			wantBodyPart: `"id":"hits"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			handler.GetMetricsByNameHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBodyPart != "" {
				assert.Contains(t, rr.Body.String(), tt.wantBodyPart)
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			}
		})
	}
}

func TestMetricsHandlerSetMetricDataHandle(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		params     map[string]string
		mock       func(*mocks.MockService)
		wantStatus int
	}{
		{
			name:       "rejects wrong method",
			method:     http.MethodGet,
			params:     map[string]string{"metricType": models.Counter, "metricName": "hits", "metricValue": "1"},
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "returns not found for missing metric name",
			method:     http.MethodPost,
			params:     map[string]string{"metricType": models.Counter, "metricName": "", "metricValue": "1"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "saves metric",
			method: http.MethodPost,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits", "metricValue": "1"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "1").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "returns bad request for service error",
			method: http.MethodPost,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits", "metricValue": "1"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "1").Return(assert.AnError)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(tt.method, "/", nil)
			req = withRouteParams(req, tt.params)
			rr := httptest.NewRecorder()

			handler.SetMetricDataHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMetricsHandlerGetMetricDataHandle(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		params     map[string]string
		mock       func(*mocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "rejects wrong method",
			method:     http.MethodPost,
			params:     map[string]string{"metricType": models.Counter, "metricName": "hits"},
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "returns not found for missing params",
			method:     http.MethodGet,
			params:     map[string]string{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "returns not found for service error",
			method: http.MethodGet,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{}, assert.AnError)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "returns counter value",
			method: http.MethodGet,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{
					ID:    "hits",
					MType: models.Counter,
					Delta: ptrInt64(5),
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "5",
		},
		{
			name:   "returns gauge value",
			method: http.MethodGet,
			params: map[string]string{"metricType": models.Gauge, "metricName": "load"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(models.Gauge, "load").Return(models.Metrics{
					ID:    "load",
					MType: models.Gauge,
					Value: ptrFloat64(1.25),
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "1.25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(tt.method, "/", nil)
			req = withRouteParams(req, tt.params)
			rr := httptest.NewRecorder()

			handler.GetMetricDataHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, rr.Body.String())
			}
		})
	}
}

func TestNewPageHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates page handler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mocks.NewMockStorage(ctrl)
			handler := NewPageHandler(storage)

			require.NotNil(t, handler)
			require.NotNil(t, handler.tpl)
		})
	}
}

func TestMetricsHandlerServePage(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		mock         func(*mocks.MockService)
		wantStatus   int
		wantBodyPart string
	}{
		{
			name:       "rejects wrong method",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "renders metrics page",
			method: http.MethodGet,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetAll().Return([]models.Metrics{{ID: "hits", MType: models.Counter, Delta: ptrInt64(1)}})
			},
			wantStatus:   http.StatusOK,
			wantBodyPart: "hits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(tt.method, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServePage(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBodyPart != "" {
				assert.Contains(t, rr.Body.String(), tt.wantBodyPart)
				assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
			}
		})
	}
}

func TestMetricsHandlerPing(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		mock       func(*mocks.MockService)
		wantStatus int
	}{
		{
			name:       "rejects wrong method",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "returns ping error",
			method: http.MethodGet,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().Ping(gomock.Any()).Return(assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "returns ok",
			method: http.MethodGet,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().Ping(gomock.Any()).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(tt.method, "/ping", nil)
			rr := httptest.NewRecorder()

			handler.Ping(context.Background()).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMetricsHandlerSetMetricsDataHandle(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		mock       func(*mocks.MockService)
		wantStatus int
	}{
		{
			name:       "rejects wrong method",
			method:     http.MethodGet,
			body:       `[]`,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "returns bad request for invalid json",
			method:     http.MethodPost,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "accepts single metric payload",
			method: http.MethodPost,
			body:   `{"id":"hits","type":"counter","delta":3}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "3").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "aggregates batch metrics",
			method: http.MethodPost,
			body:   `[{"id":"hits","type":"counter","delta":2},{"id":"hits","type":"counter","delta":3},{"id":"load","type":"gauge","value":1.5},{"id":"load","type":"gauge","value":2.5}]`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "5").Return(nil)
				svc.EXPECT().SetData(models.Gauge, "load", "2.5").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(tt.method, "/updates", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			handler.SetMetricsDataHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMetricsHandlerSetDataWithRetry(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(*mocks.MockService)
		wantErr bool
	}{
		{
			name: "returns nil on first try",
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "1").Return(nil)
			},
		},
		{
			name: "retries retriable error and succeeds",
			mock: func(svc *mocks.MockService) {
				gomock.InOrder(
					svc.EXPECT().SetData(models.Counter, "hits", "1").Return(&pgconn.PgError{Code: pgerrcode.ConnectionFailure}),
					svc.EXPECT().SetData(models.Counter, "hits", "1").Return(nil),
				)
			},
		},
		{
			name: "returns non retriable error without retry",
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "1").Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			tt.mock(svc)

			handler := NewMetricsHandler(svc)
			err := handler.setDataWithRetry(models.Counter, "hits", "1")

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMetricsHandlerSaveMetric(t *testing.T) {
	tests := []struct {
		name    string
		metric  models.Metrics
		strict  bool
		mock    func(*mocks.MockService)
		wantErr bool
	}{
		{
			name:   "saves counter metric",
			metric: models.Metrics{ID: "hits", MType: models.Counter, Delta: ptrInt64(5)},
			strict: true,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(models.Counter, "hits", "5").Return(nil)
			},
		},
		{
			name:    "strict validation returns error",
			metric:  models.Metrics{ID: "hits", MType: models.Counter},
			strict:  true,
			wantErr: true,
		},
		{
			name:   "non strict skips invalid metric",
			metric: models.Metrics{ID: "hits", MType: "unknown"},
			strict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			if tt.mock != nil {
				tt.mock(svc)
			}

			handler := NewMetricsHandler(svc)
			err := handler.saveMetric(tt.metric, tt.strict)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetMetricValue(t *testing.T) {
	tests := []struct {
		name      string
		metric    models.Metrics
		strict    bool
		wantValue string
		wantSkip  bool
		wantErr   bool
	}{
		{
			name:      "returns counter value",
			metric:    models.Metrics{MType: models.Counter, Delta: ptrInt64(5)},
			strict:    true,
			wantValue: "5",
		},
		{
			name:      "returns gauge value",
			metric:    models.Metrics{MType: models.Gauge, Value: ptrFloat64(1.5)},
			strict:    true,
			wantValue: "1.5",
		},
		{
			name:    "strict counter without delta returns error",
			metric:  models.Metrics{MType: models.Counter},
			strict:  true,
			wantErr: true,
		},
		{
			name:     "non strict unsupported metric is skipped",
			metric:   models.Metrics{MType: "other"},
			strict:   false,
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, skip, err := getMetricValue(tt.metric, tt.strict)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantSkip, skip)
		})
	}
}

func TestMetricValidationError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{name: "returns message", message: "bad metric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := metricValidationError{message: tt.message}
			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestMetricsHandlerGetMetricsByNameHandleResponseJSON(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "response is valid json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{
				ID:    "hits",
				MType: models.Counter,
				Delta: ptrInt64(7),
			}, nil)

			handler := NewMetricsHandler(svc)
			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"hits","type":"counter"}`))
			rr := httptest.NewRecorder()

			handler.GetMetricsByNameHandle().ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var got models.Metrics
			require.NoError(t, json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&got))
			assert.Equal(t, "hits", got.ID)
		})
	}
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Nakohartum/practicum-metrics/internal/audit"
	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/service"
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

type recordingAuditObserver struct {
	events []audit.Event
}

func (o *recordingAuditObserver) Notify(ctx context.Context, event audit.Event) error {
	o.events = append(o.events, event)
	return nil
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
			name: "returns bad request for missing delta in strict mode",
			body: `{"id":"hits","type":"counter"}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
					{ID: "hits", MType: models.Counter},
				}).Return(service.ErrCounterDeltaRequired)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "saves valid metric",
			body: `{"id":"hits","type":"counter","delta":3}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
					{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
				}).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "returns bad request for service failure",
			body: `{"id":"hits","type":"counter","delta":3}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
					{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
				}).Return(assert.AnError)
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
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			handler.UpdateMetricsDataHandle().ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestMetricsHandlerUpdateMetricsDataHandleNotifiesAudit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockService(ctrl)
	svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
		{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
	}).Return(nil)

	observer := &recordingAuditObserver{}
	handler := NewMetricsHandler(svc, audit.NewAuditor(observer))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"id":"hits","type":"counter","delta":3}`))
	req.RemoteAddr = "192.168.0.42:12345"
	rr := httptest.NewRecorder()

	handler.UpdateMetricsDataHandle().ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, observer.events, 1)
	assert.Equal(t, []string{"hits"}, observer.events[0].Metrics)
	assert.Equal(t, "192.168.0.42", observer.events[0].IPAddress)
	assert.NotZero(t, observer.events[0].Ts)
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
				svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{}, assert.AnError)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "returns metric json",
			body: `{"id":"hits","type":"counter"}`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{
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

func TestMetricsHandlerGetMetricsByNameHandleDoesNotNotifyAudit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockService(ctrl)
	svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{
		ID:    "hits",
		MType: models.Counter,
		Delta: ptrInt64(5),
	}, nil)

	observer := &recordingAuditObserver{}
	handler := NewMetricsHandler(svc, audit.NewAuditor(observer))
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"hits","type":"counter"}`))
	rr := httptest.NewRecorder()

	handler.GetMetricsByNameHandle().ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, observer.events)
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
				svc.EXPECT().SetData(gomock.Any(), models.Counter, "hits", "1").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "returns bad request for service error",
			method: http.MethodPost,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits", "metricValue": "1"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetData(gomock.Any(), models.Counter, "hits", "1").Return(assert.AnError)
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
				svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{}, assert.AnError)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "returns counter value",
			method: http.MethodGet,
			params: map[string]string{"metricType": models.Counter, "metricName": "hits"},
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{
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
				svc.EXPECT().GetData(gomock.Any(), models.Gauge, "load").Return(models.Metrics{
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

			svc := mocks.NewMockService(ctrl)
			handler := NewPageHandler(svc)

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
				svc.EXPECT().GetAll(gomock.Any()).Return([]models.Metrics{{ID: "hits", MType: models.Counter, Delta: ptrInt64(1)}})
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

			handler.Ping().ServeHTTP(rr, req)

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
				svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
					{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
				}).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "delegates batch metrics to service",
			method: http.MethodPost,
			body:   `[{"id":"hits","type":"counter","delta":2},{"id":"hits","type":"counter","delta":3},{"id":"load","type":"gauge","value":1.5},{"id":"load","type":"gauge","value":2.5}]`,
			mock: func(svc *mocks.MockService) {
				svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
					{ID: "hits", MType: models.Counter, Delta: ptrInt64(2)},
					{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
					{ID: "load", MType: models.Gauge, Value: ptrFloat64(1.5)},
					{ID: "load", MType: models.Gauge, Value: ptrFloat64(2.5)},
				}).Return(nil)
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

func TestMetricsHandlerSetMetricsDataHandleNotifiesSubmittedMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockService(ctrl)
	svc.EXPECT().SetDataUsingMetrics(gomock.Any(), []models.Metrics{
		{ID: "hits", MType: models.Counter, Delta: ptrInt64(3)},
		{ID: "load", MType: models.Gauge, Value: ptrFloat64(1.5)},
	}).Return(nil)

	observer := &recordingAuditObserver{}
	handler := NewMetricsHandler(svc, audit.NewAuditor(observer))
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(`[{"id":"hits","type":"counter","delta":3},{"id":"load","type":"gauge","value":1.5}]`))
	rr := httptest.NewRecorder()

	handler.SetMetricsDataHandle().ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, observer.events, 1)
	assert.Equal(t, []string{"hits", "load"}, observer.events[0].Metrics)
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
			svc.EXPECT().GetData(gomock.Any(), models.Counter, "hits").Return(models.Metrics{
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

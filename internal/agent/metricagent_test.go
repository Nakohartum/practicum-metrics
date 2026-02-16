package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetGaugeMetric(t *testing.T) {

	tests := []struct {
		name       string 
		initialGaugeMetrics map[string]float64
		metricName string
		value      float64
		expected   float64
		times      int
	}{
		{
			name: "Positive case #1",
			initialGaugeMetrics: map[string]float64{
				"testMetric":2.4,
			},
			metricName: "testMetric",
			value: 3.5,
			expected: 3.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ma := NewAgentMetrics(0, 0)
			ma.gaugeMetrics = tt.initialGaugeMetrics
			ma.setGaugeMetric(tt.metricName, tt.value)
			if ma.gaugeMetrics[tt.metricName] != tt.expected {
				t.Errorf("Not correct value for key: %s. Value: %f. Expected: %f", tt.metricName, ma.gaugeMetrics[tt.metricName], tt.expected)
			}
		})
	}
}

func TestSendGaugeMetrics(t *testing.T){
	type wantReq struct{
		name string
		value string
	}
	tests := []struct{
		name string
		metrics map[string]float64
		want []wantReq
	}{
		{
			name: "two metrics",
			metrics: map[string]float64{
				"alloc":     1.5,
				"heapAlloc": 42,
			},
			want: []wantReq{
				{name: "alloc", value: "1.5"},
				{name: "heapAlloc", value: "42"},
			},
		},
		{
			name: "single metric",
			metrics: map[string]float64{
				"randomValue": 0.25,
			},
			want: []wantReq{
				{name: "randomValue", value: "0.25"},
			},
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T){
			ma := NewAgentMetrics(0,0)
			ma.gaugeMetrics = tt.metrics
			
			var gotPaths []string
			var gotMethods []string
			var gotContentTypes []string

			// создаем сервер
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
				gotPaths = append(gotPaths, r.URL.Path)
				gotMethods = append(gotMethods, r.Method)
				gotContentTypes = append(gotContentTypes, r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)

				_,_ = w.Write([]byte("ok"))
			}))

			defer ts.Close()

			ma.client.SetTransport(ts.Client().Transport)

			ma.sendGaugeMetrics(ts.URL)

			// проверяем все ли пути прошли
			if len(gotPaths) != len(tt.want){
				t.Fatalf("requests=%d. want=%d", len(gotPaths), len(tt.want))
			}


			// смотрим соответствие каждого запроса
			for i := range gotPaths {
				if gotMethods[i] != http.MethodPost{
					t.Fatalf("method=%s. want=%s", gotMethods[i], http.MethodPost)
				}

				if gotContentTypes[i] != "application/json"{
					t.Fatalf("Content-Type=%s, want=%s", gotContentTypes[i], "application/json")
				}

				if !strings.HasPrefix(gotPaths[i], "/update"){
					t.Fatalf("unexpected path:%s", gotPaths[i])
				}
			}
		})
	}
}

func TestSendCounterMetrics(t *testing.T){
	type wantReq struct{
		name string
		value string
	}
	tests := []struct{
		name string
		metrics map[string]int64
		want []wantReq
	}{
		{
			name: "two metrics",
			metrics: map[string]int64{
				"counterOne":     1,
				"counterTwo": 1,
			},
			want: []wantReq{
				{name: "counterOne", value: "1"},
				{name: "counterTwo", value: "1"},
			},
		},
		{
			name: "single metric",
			metrics: map[string]int64{
				"randomCounter": 1,
			},
			want: []wantReq{
				{name: "randomCounter", value: "1"},
			},
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T){
			ma := NewAgentMetrics(0,0)
			ma.counterMetrics = tt.metrics
			
			var gotPaths []string
			var gotMethods []string
			var gotContentTypes []string

			// создаем сервер
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
				gotPaths = append(gotPaths, r.URL.Path)
				gotMethods = append(gotMethods, r.Method)
				gotContentTypes = append(gotContentTypes, r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)

				_,_ = w.Write([]byte("ok"))
			}))

			defer ts.Close()

			ma.client.SetTransport(ts.Client().Transport)

			ma.sendCounterMetrics(ts.URL)

			// проверяем все ли пути прошли
			if len(gotPaths) != len(tt.want){
				t.Fatalf("requests=%d. want=%d", len(gotPaths), len(tt.want))
			}


			// смотрим соответствие каждого запроса
			for i := range gotPaths {
				if gotMethods[i] != http.MethodPost{
					t.Fatalf("method=%s. want=%s", gotMethods[i], http.MethodPost)
				}

				if gotContentTypes[i] != "application/json"{
					t.Fatalf("Content-Type=%s, want=%s", gotContentTypes[i], "application/json")
				}

				if !strings.HasPrefix(gotPaths[i], "/update"){
					t.Fatalf("unexpected path:%s", gotPaths[i])
				}
			}
		})
	}
}
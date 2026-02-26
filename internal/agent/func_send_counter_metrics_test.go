package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
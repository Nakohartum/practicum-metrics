package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHeader(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "stores 200", status: http.StatusOK},
		{name: "stores 404", status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			lw := &loggingResponseWriter{ResponseWriter: rr, status: 200}
			lw.WriteHeader(tt.status)
			assert.Equal(t, tt.status, lw.status)
			assert.Equal(t, tt.status, rr.Code)
		})
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "tracks response size", body: "hello"},
		{name: "tracks empty body", body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			lw := &loggingResponseWriter{ResponseWriter: rr, status: 200}
			n, err := lw.Write([]byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, len(tt.body), n)
			assert.Equal(t, len(tt.body), lw.size)
			assert.Equal(t, tt.body, rr.Body.String())
		})
	}
}

func TestAttachLoggingToResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "preserves success response", statusCode: http.StatusOK, body: "ok"},
		{name: "preserves error response", statusCode: http.StatusBadRequest, body: "bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})

			wrapped := AttachLoggingToResponse(h)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			assert.Equal(t, tt.statusCode, rr.Code)
			assert.Equal(t, tt.body, rr.Body.String())
		})
	}
}

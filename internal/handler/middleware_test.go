package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipBody(t *testing.T, payload string) io.Reader {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(payload))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return &buf
}

func ungzipBytes(t *testing.T, payload []byte) string {
	zr, err := gzip.NewReader(bytes.NewReader(payload))
	require.NoError(t, err)
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	return string(decoded)
}

func TestGetZippedDataMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		contentEncoding string
		body            io.Reader
		wantStatus      int
		wantBody        string
	}{
		{
			name:            "passes plain body when no gzip",
			contentEncoding: "",
			body:            bytes.NewBufferString("plain"),
			wantStatus:      http.StatusOK,
			wantBody:        "plain",
		},
		{
			name:            "decompresses gzip body",
			contentEncoding: "gzip",
			body:            gzipBody(t, "compressed"),
			wantStatus:      http.StatusOK,
			wantBody:        "compressed",
		},
		{
			name:            "returns bad request for invalid gzip",
			contentEncoding: "gzip",
			body:            bytes.NewBufferString("not-gzip"),
			wantStatus:      http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
			})

			req := httptest.NewRequest(http.MethodPost, "/", tt.body)
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			rr := httptest.NewRecorder()

			GetZippedDataMiddleware(next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, rr.Body.String())
			}
		})
	}
}

func TestGiveZippedDataMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		contentType    string
		wantGzip       bool
	}{
		{
			name:           "gzip response when accepted and json",
			acceptEncoding: "gzip",
			contentType:    "application/json",
			wantGzip:       true,
		},
		{
			name:           "plain response when no accept gzip",
			acceptEncoding: "",
			contentType:    "application/json",
			wantGzip:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			rr := httptest.NewRecorder()

			GiveZippedDataMiddleware(next).ServeHTTP(rr, req)

			if tt.wantGzip {
				assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))
				assert.Equal(t, `{"ok":true}`, ungzipBytes(t, rr.Body.Bytes()))
				return
			}
			assert.Empty(t, rr.Header().Get("Content-Encoding"))
			assert.Equal(t, `{"ok":true}`, rr.Body.String())
		})
	}
}

func TestSaveAfterPostMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		status    int
		saveCalls int
	}{
		{name: "post success triggers save", method: http.MethodPost, status: http.StatusOK, saveCalls: 1},
		{name: "post error does not trigger save", method: http.MethodPost, status: http.StatusInternalServerError, saveCalls: 0},
		{name: "get does not trigger save", method: http.MethodGet, status: http.StatusOK, saveCalls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().SaveAllData().Return(nil).Times(tt.saveCalls)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			})

			h := SaveAfterPostMiddleware(svc)(next)
			req := httptest.NewRequest(tt.method, "/", nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.status, rr.Code)
		})
	}
}

func TestHashMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		body       string
		hash       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "passes through when key is empty",
			key:        "",
			body:       "payload",
			wantStatus: http.StatusOK,
			wantBody:   "payload",
		},
		{
			name:       "returns bad request for missing hash",
			key:        "secret",
			body:       "payload",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request for invalid hash",
			key:        "secret",
			body:       "payload",
			hash:       "bad",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "passes valid hash",
			key:        "secret",
			body:       "payload",
			wantStatus: http.StatusOK,
			wantBody:   "payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
			})

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			if tt.key != "" && tt.hash == "" && tt.wantStatus == http.StatusOK {
				req.Header.Set("HashSHA256", calculateHash(tt.body, tt.key))
			}
			if tt.hash != "" {
				req.Header.Set("HashSHA256", tt.hash)
			}
			rr := httptest.NewRecorder()

			HashMiddleware(tt.key)(next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, rr.Body.String())
			}
		})
	}
}

func calculateHash(body, key string) string {
	data := make([]byte, 0, len(body)+len(key))
	data = append(data, body...)
	data = append(data, key...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

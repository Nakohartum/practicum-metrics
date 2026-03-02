package handler_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Nakohartum/practicum-metrics/internal/handler"
	"github.com/Nakohartum/practicum-metrics/internal/mocks"
)

// =========================
// helpers
// =========================

func gzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(b)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func ungzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	defer zr.Close()

	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

// =========================
// GetZippedDataMiddleware tests
// =========================

func TestGetZippedDataMiddleware_Table(t *testing.T) {
	type tc struct {
		name           string
		contentEnc     string
		body           []byte
		wantStatus     int
		wantDownstream []byte
	}

	cases := []tc{
		{
			name:           "no_gzip_pass_through",
			contentEnc:     "",
			body:           []byte("plain"),
			wantStatus:     http.StatusOK,
			wantDownstream: []byte("plain"),
		},
		{
			name:           "gzip_ok_decompress",
			contentEnc:     "gzip",
			body:           gzipBytes(t, []byte(`{"a":1}`)),
			wantStatus:     http.StatusOK,
			wantDownstream: []byte(`{"a":1}`),
		},
		{
			name:       "gzip_bad_returns_400",
			contentEnc: "gzip",
			body:       []byte("not-a-gzip-stream"),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				got = b
				w.WriteHeader(http.StatusOK)
			})

			mw := handler.GetZippedDataMiddleware(next)

			req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(tt.body))
			if tt.contentEnc != "" {
				req.Header.Set("Content-Encoding", tt.contentEnc)
			}
			rr := httptest.NewRecorder()

			mw.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantDownstream, got)
				// middleware удаляет Content-Encoding перед next
				assert.Empty(t, req.Header.Get("Content-Encoding"))
			}
		})
	}
}

// =========================
// GiveZippedDataMiddleware tests
// =========================

func TestGiveZippedDataMiddleware_Table(t *testing.T) {
	type tc struct {
		name              string
		acceptEnc         string
		contentType       string
		payload           []byte
		wantGzip          bool
		wantStatus        int
		expectedPlainBody []byte
	}

	cases := []tc{
		{
			name:              "no_accept_encoding_no_gzip",
			acceptEnc:         "",
			contentType:       "application/json",
			payload:           []byte(`{"ok":true}`),
			wantGzip:          false,
			wantStatus:        http.StatusOK,
			expectedPlainBody: []byte(`{"ok":true}`),
		},
		{
			name:        "accept_gzip_json_gzipped",
			acceptEnc:   "gzip",
			contentType: "application/json",
			payload:     []byte(`{"ok":true}`),
			wantGzip:    true,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "accept_gzip_html_gzipped",
			acceptEnc:   "gzip, deflate",
			contentType: "text/html; charset=utf-8",
			payload:     []byte(`<h1>hi</h1>`),
			wantGzip:    true,
			wantStatus:  http.StatusOK,
		},
		{
			name:              "accept_gzip_but_content_type_not_supported_no_gzip",
			acceptEnc:         "gzip",
			contentType:       "text/plain",
			payload:           []byte("hello"),
			wantGzip:          false,
			wantStatus:        http.StatusOK,
			expectedPlainBody: []byte("hello"),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// важно: Content-Type должен быть выставлен до WriteHeader/Write
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tt.payload)
			})

			mw := handler.GiveZippedDataMiddleware(next)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			if tt.acceptEnc != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEnc)
			}
			rr := httptest.NewRecorder()

			mw.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)

			body := rr.Body.Bytes()
			enc := rr.Header().Get("Content-Encoding")

			if tt.wantGzip {
				require.Equal(t, "gzip", enc)
				plain := ungzipBytes(t, body)
				assert.Equal(t, tt.payload, plain)
			} else {
				assert.NotEqual(t, "gzip", enc)
				assert.Equal(t, tt.expectedPlainBody, body)
			}
		})
	}
}

// =========================
// SaveAfterPostMiddleware tests (gomock)
// =========================


func TestSaveAfterPostMiddleware_Table(t *testing.T) {
	type tc struct {
		name           string
		method         string
		handlerStatus  int
		expectSaveCall bool
	}

	cases := []tc{
		{"POST_200_calls_save", http.MethodPost, 200, true},
		{"POST_201_calls_save", http.MethodPost, 201, true},
		{"POST_204_calls_save", http.MethodPost, 204, true},
		{"POST_400_no_save", http.MethodPost, 400, false},
		{"POST_500_no_save", http.MethodPost, 500, false},
		{"GET_200_no_save", http.MethodGet, 200, false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			saver := mocks.NewMockService(ctrl)

			require.NotNil(t, saver, "замени saver на сгенеренный gomock-мок (см. комментарии выше)")

			// ожидания
			if tt.expectSaveCall {
				// gomock expectation:
				saver.EXPECT().SaveAllData().Return(nil).Times(1)
			} else {
				saver.EXPECT().SaveAllData().Times(0)
			}

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				_, _ = w.Write([]byte("ok"))
			})

			mw := handler.SaveAfterPostMiddleware(saver /* тут будет твой тип */)(next)

			req := httptest.NewRequest(tt.method, "http://example.com/", nil)
			rr := httptest.NewRecorder()

			mw.ServeHTTP(rr, req)

			require.Equal(t, tt.handlerStatus, rr.Code)
		})
	}
}
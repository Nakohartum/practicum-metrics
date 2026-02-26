package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// helpers
func gzipBytes(t *testing.T, in []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(in)
	if err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func ungzipBytes(t *testing.T, in []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer zr.Close()

	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("ungzip read: %v", err)
	}
	return out
}

func readAllBody(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	return string(b)
}

func TestGetZippedDataMiddleware(t *testing.T) {
	type testCase struct {
		name            string
		contentEncoding string
		body            []byte
		wantBody        string
		wantStatus      int
		wantErrStatus   int // 0 if no error expected
	}

	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data)
	})

	tests := []testCase{
		{
			name:            "PassThrough_WhenNoGzip",
			contentEncoding: "",
			body:            []byte("plain"),
			wantBody:        "plain",
			wantStatus:      http.StatusOK,
		},
		{
			name:            "UnzipsBody_WhenGzipEncoding",
			contentEncoding: "gzip",
			body:            gzipBytes(t, []byte("hello gzip")),
			wantBody:        "hello gzip",
			wantStatus:      http.StatusOK,
		},
		{
			name:            "BadRequest_WhenInvalidGzipBody",
			contentEncoding: "gzip",
			body:            []byte("not-gzip-at-all"),
			wantErrStatus:   http.StatusBadRequest,
		},
		{
			name:            "UnzipsBody_WhenHeaderContainsGzipToken",
			contentEncoding: "br, gzip",
			body:            gzipBytes(t, []byte("token gzip")),
			wantBody:        "token gzip",
			wantStatus:      http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run("TestGetZippedDataMiddleware_"+tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			rr := httptest.NewRecorder()
			h := GetZippedDataMiddleware(echoHandler)
			h.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			if tt.wantErrStatus != 0 {
				if res.StatusCode != tt.wantErrStatus {
					t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantErrStatus)
				}
				return
			}

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}

			got := readAllBody(t, res.Body)
			if got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}

			// middleware должен удалить Content-Encoding, чтобы дальше хендлеры не думали, что тело ещё сжато
			if v := req.Header.Get("Content-Encoding"); v != "" {
				t.Fatalf("request header Content-Encoding not cleared: %q", v)
			}
		})
	}
}

func TestGiveZippedDataMiddleware(t *testing.T) {
	type testCase struct {
		name           string
		acceptEncoding string
		contentType    string
		handlerBody    string

		wantGzip      bool
		wantPlainBody string
	}

	tests := []testCase{
		{
			name:           "PassThrough_WhenClientNotAcceptGzip",
			acceptEncoding: "",
			contentType:    "application/json",
			handlerBody:    `{"ok":true}`,
			wantGzip:       false,
			wantPlainBody:  `{"ok":true}`,
		},
		{
			name:           "Gzips_WhenAcceptGzip_AndJSON",
			acceptEncoding: "gzip",
			contentType:    "application/json",
			handlerBody:    `{"ok":true}`,
			wantGzip:       true,
			wantPlainBody:  `{"ok":true}`,
		},
		{
			name:           "Gzips_WhenAcceptGzip_AndHTML",
			acceptEncoding: "gzip",
			contentType:    "text/html; charset=utf-8",
			handlerBody:    "<h1>hi</h1>",
			wantGzip:       true,
			wantPlainBody:  "<h1>hi</h1>",
		},
		{
			name:           "DoesNotGzip_WhenAcceptGzip_ButContentTypeOther",
			acceptEncoding: "gzip",
			contentType:    "text/plain",
			handlerBody:    "plain text",
			wantGzip:       false,
			wantPlainBody:  "plain text",
		},
		{
			name:           "GzipTokenInAcceptEncoding",
			acceptEncoding: "br, gzip",
			contentType:    "application/json",
			handlerBody:    `{"x":1}`,
			wantGzip:       true,
			wantPlainBody:  `{"x":1}`,
		},
	}

	for _, tt := range tests {
		t.Run("TestGiveZippedDataMiddleware_"+tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				_, _ = w.Write([]byte(tt.handlerBody))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			rr := httptest.NewRecorder()
			h := GiveZippedDataMiddleware(next)
			h.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			gotBytes, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			enc := res.Header.Get("Content-Encoding")
			if tt.wantGzip {
				if enc != "gzip" {
					t.Fatalf("Content-Encoding = %q, want %q", enc, "gzip")
				}
				plain := string(ungzipBytes(t, gotBytes))
				if plain != tt.wantPlainBody {
					t.Fatalf("unzipped body = %q, want %q", plain, tt.wantPlainBody)
				}
			} else {
				if enc != "" {
					t.Fatalf("Content-Encoding = %q, want empty", enc)
				}
				if string(gotBytes) != tt.wantPlainBody {
					t.Fatalf("body = %q, want %q", string(gotBytes), tt.wantPlainBody)
				}
			}
		})
	}
}

type saverMock struct {
	calls int32
	err   error
}

func (s *saverMock) SaveData() error {
	atomic.AddInt32(&s.calls, 1)
	return s.err
}

func TestSaveAfterPostMiddleware(t *testing.T) {
	type testCase struct {
		name       string
		method     string
		handlerSC  int
		wantSaves  int32
		wantNext   int32
	}

	tests := []testCase{
		{
			name:      "DoesNotSave_OnGET",
			method:    http.MethodGet,
			handlerSC: http.StatusOK,
			wantSaves: 0,
			wantNext:  1,
		},
		{
			name:      "Saves_OnPOST_SuccessStatus",
			method:    http.MethodPost,
			handlerSC: http.StatusOK,
			wantSaves: 1,
			// ОЖИДАНИЕ: next вызывается 1 раз
			// ФАКТ в твоём коде сейчас: будет 2 раза (баг)
			wantNext:  1,
		},
		{
			name:      "DoesNotSave_OnPOST_ErrorStatus",
			method:    http.MethodPost,
			handlerSC: http.StatusBadRequest,
			wantSaves: 0,
			// То же ожидание: 1 вызов next
			wantNext:  1,
		},
		{
			name:      "DoesNotSave_OnPOST_ServerError",
			method:    http.MethodPost,
			handlerSC: http.StatusInternalServerError,
			wantSaves: 0,
			wantNext:  1,
		},
	}

	for _, tt := range tests {
		t.Run("TestSaveAfterPostMiddleware_"+tt.name, func(t *testing.T) {
			var nextCalls int32
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&nextCalls, 1)
				w.WriteHeader(tt.handlerSC)
				_, _ = w.Write([]byte("ok"))
			})

			saver := &saverMock{}
			mw := SaveAfterPostMiddleware(saver)

			req := httptest.NewRequest(tt.method, "/test", nil)
			rr := httptest.NewRecorder()

			mw(next).ServeHTTP(rr, req)

			if got := atomic.LoadInt32(&saver.calls); got != tt.wantSaves {
				t.Fatalf("SaveData calls = %d, want %d", got, tt.wantSaves)
			}
			if got := atomic.LoadInt32(&nextCalls); got != tt.wantNext {
				t.Fatalf("next handler calls = %d, want %d", got, tt.wantNext)
			}
		})
	}
}

func TestGiveZippedDataMiddleware_ContentTypeMustBeSetBeforeWrite(t *testing.T) {
	// Этот тест показывает важную особенность твоей реализации:
	// gzip включается только если Content-Type уже выставлен к моменту WriteHeader.
	// Поэтому если хендлер НЕ выставит Content-Type, то gzip не включится.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Type не ставим
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GiveZippedDataMiddleware(next).ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if got := res.Header.Get("Content-Encoding"); strings.EqualFold(got, "gzip") {
		t.Fatalf("unexpected gzip without Content-Type, Content-Encoding=%q", got)
	}
}
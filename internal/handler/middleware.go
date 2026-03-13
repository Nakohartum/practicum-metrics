package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/Nakohartum/practicum-metrics/internal/service"
)

type gZipWriter struct {
	http.ResponseWriter
	zw          *gzip.Writer
	gzipEnabled bool
	wroteHeader bool
}

func newGzipWriter(w http.ResponseWriter) *gZipWriter {
	return &gZipWriter{ResponseWriter: w}
}

func (w *gZipWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.gzipEnabled && w.zw != nil {
		return w.zw.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gZipWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	if strings.Contains(w.Header().Get("Content-Type"), "application/json") || strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		w.gzipEnabled = true
		w.Header().Set("Content-Encoding", "gzip")
		w.zw = gzip.NewWriter(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gZipWriter) Close() error {
	if w.gzipEnabled && w.zw != nil {
		return w.zw.Close()
	}
	return nil
}

type gZipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newGZipReader(r io.ReadCloser) (*gZipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &gZipReader{r: r, zr: zr}, nil
}

func (gr *gZipReader) Read(p []byte) (int, error) {
	return gr.zr.Read(p)
}

func (gr *gZipReader) Close() error {
	err := gr.zr.Close()
	if err != nil {
		return err
	}
	return gr.r.Close()
}

func GetZippedDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		cw, err := newGZipReader(r.Body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		r.Body = cw
		defer cw.Close()
		r.Header.Del("Content-Encoding")
		next.ServeHTTP(w, r)
	})
}

func GiveZippedDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		cw := newGzipWriter(w)

		defer cw.Close()

		next.ServeHTTP(cw, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func SaveAfterPostMiddleware(saver service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
				next.ServeHTTP(sw, r)
				if sw.statusCode < 400 {
					_ = saver.SaveAllData()
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			headerHash := r.Header.Get("HashSHA256")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			if headerHash == "" {
				http.Error(w, "missing hash", http.StatusBadRequest)
				return
			}

			data := make([]byte, 0, len(body)+len(key))
			data = append(data, body...)
			data = append(data, key...)
			hash := sha256.Sum256(data)
			hashHex := hex.EncodeToString(hash[:])

			if hashHex != headerHash {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

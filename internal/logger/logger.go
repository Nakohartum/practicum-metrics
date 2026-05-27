package logger

import (
	"context"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var logger, _ = zap.NewDevelopment()

type loggerKey string

var timeKey = loggerKey("start")

var sugar = *logger.Sugar()

// AttachLoggingToRequest adds outgoing request logging hooks to a Resty client.
func AttachLoggingToRequest(r *resty.Client) {
	r.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		ctx := context.WithValue(r.Context(), timeKey, time.Now())
		r.SetContext(ctx)
		return nil
	})

	r.OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
		start, _ := r.Request.Context().Value(timeKey).(time.Time)
		duration := time.Since(start)

		uri := r.Request.RawRequest.RequestURI

		sugar.Infow("outgoing request", "uri", uri, "method", r.Request.Method, "duration", duration)
		return nil
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

// AttachLoggingToResponse wraps an HTTP handler with response logging.
func AttachLoggingToResponse(h http.Handler) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			status:         200,
			size:           0,
		}
		h.ServeHTTP(lrw, r)

		sugar.Infow("response", "status code", lrw.status, "size of response", lrw.size)
	}

	return http.HandlerFunc(logFn)
}

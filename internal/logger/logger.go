package logger

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var logger, err = zap.NewDevelopment()

var sugar = *logger.Sugar()

func AttachLoggingToRequest(r *resty.Client) {
	r.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		ctx := context.WithValue(r.Context(), "start", time.Now())
		r.SetContext(ctx)
		return nil
	})

	r.OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
		start, _ := r.Request.Context().Value("start").(time.Time)
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


func AttachLoggingToResponse(h http.Handler) http.HandlerFunc {
	logFn := func (w http.ResponseWriter, r *http.Request)  {
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			status: 200,
			size: 0,
		}
		h.ServeHTTP(lrw, r)

		res,_ := io.ReadAll(r.Body)
		size := len(res)

		sugar.Infow("response", "status code", r.Response.StatusCode, "size of response", size)
	}

	return http.HandlerFunc(logFn)
}
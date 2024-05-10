package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func LoggerMiddleware(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		respData := &responseData{
			statusCode: 0,
			size:       0,
		}
		writer := logResponseWriter{
			ResponseWriter: w,
			responseData:   respData,
		}
		h.ServeHTTP(&writer, r)

		duration := time.Since(start)

		zap.L().Info(
			"got incoming HTTP request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", duration),
			zap.Int("status", respData.statusCode),
			zap.Int("size", respData.size),
		)
	}

	return http.HandlerFunc(logFn)
}
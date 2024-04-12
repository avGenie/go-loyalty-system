package http

import "net/http"

type (
	responseData struct {
		statusCode int
		size       int
	}

	logResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (w *logResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseData.size += size

	return size, err
}

func (w *logResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.statusCode = statusCode
}

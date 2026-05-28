package server

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

type contextKey string

const requestIDKey contextKey = "request_id"

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return requestID
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request completed",
			"request_id", requestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start),
		)

	})
}

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				logger.Error(
					"panic recovered",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", v,
				)

				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func requestIDMiddlware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)

		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

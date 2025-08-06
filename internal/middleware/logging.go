// Package middleware provides HTTP middleware functionality.
package middleware

import (
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/reservia/api/pkg/logger"
)

// LoggingMiddleware provides custom HTTP request logging with path exclusions.
type LoggingMiddleware struct {
	logger        logger.Logger
	excludedPaths []string
}

// NewLoggingMiddleware creates a new logging middleware instance.
func NewLoggingMiddleware(log logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log,
		excludedPaths: []string{
			"/health",   // Exclude health checks
			"/metrics",  // Optionally exclude metrics endpoint
		},
	}
}

// Handler returns the logging middleware handler function.
func (l *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for excluded paths
		for _, path := range l.excludedPaths {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Use Chi's default logging for non-excluded paths
		logEntry := chiMiddleware.GetLogEntry(r)
		if logEntry == nil {
			// If no log entry exists, create a minimal wrapper
			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				l.logger.Info("HTTP request",
					"method", r.Method,
					"uri", r.RequestURI,
					"proto", r.Proto,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration", time.Since(start),
					"remote_addr", r.RemoteAddr,
					"user_agent", r.Header.Get("User-Agent"),
					"referer", r.Header.Get("Referer"),
				)
			}()

			next.ServeHTTP(ww, r)
			return
		}

		// Use Chi's standard logging
		next.ServeHTTP(w, r)
	})
}

// WithExcludedPaths allows customizing which paths to exclude from logging.
func (l *LoggingMiddleware) WithExcludedPaths(paths []string) *LoggingMiddleware {
	l.excludedPaths = paths
	return l
}
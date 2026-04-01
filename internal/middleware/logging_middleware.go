package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

func redactHeaders(headers http.Header) http.Header {
	redactedHeaders := make(http.Header)
	for key, values := range headers {
		if strings.EqualFold(key, "Authorization") || strings.EqualFold(key, "expo-session") {
			redactedHeaders[key] = []string{"REDACTED"}
		} else {
			redactedHeaders[key] = values
		}
	}
	return redactedHeaders
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			if err := recover(); err != nil {
				safeHeaders := redactHeaders(r.Header)
				log.Printf("Panic recovered during %s %s\nQuery: %s\nHeaders: %v\nError: %v\nStack Trace:\n%s",
					r.Method, r.RequestURI, r.URL.RawQuery, safeHeaders, err, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(recorder, r)

		if recorder.statusCode >= 500 {
			safeHeaders := redactHeaders(r.Header)
			log.Printf("Error: %s %s returned %d in %v\nQuery: %s\nHeaders: %v",
				r.Method, r.RequestURI, recorder.statusCode, time.Since(start), r.URL.RawQuery, safeHeaders)
		} else {
			log.Printf("%s %s %d %v", r.Method, r.RequestURI, recorder.statusCode, time.Since(start))
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

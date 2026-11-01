package server

import (
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

// secretParameters are the query parameters that carry a token or key,
// masked in the request log.
var secretParameters = regexp.MustCompile(`(?i)((?:api_?key|x-emby-token|token)=)[^&]+`)

// logRequests logs every request with its answer, for diagnosing an app:
// the method, the path and query with tokens masked, the status, the time
// taken, and how the app named itself. Bodies, which hold passwords, are
// never logged.
func logRequests(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		status := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(status, r)
		target := r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + secretParameters.ReplaceAllString(r.URL.RawQuery, "${1}…")
		}
		logger.Info("Request", "method", r.Method, "target", target, "status", status.code(),
			"ms", time.Since(started).Milliseconds(), "type", r.Header.Get("Content-Type"),
			"client", r.Header.Get("X-Emby-Client"), "agent", r.UserAgent())
	})
}

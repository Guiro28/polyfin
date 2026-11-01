package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/moodiness/polyfin/internal/embyapp"
)

// embyManifest is the path of the web app manifest under the Emby prefix.
// Emby for Android TV reads it right after signing in, and calls the server
// incompatible, then signs out, when it names Jellyfin, as jellyfin-web's
// does.
const embyManifest = "/web/manifest.json"

// underEmbyBase serves a request whose path starts with /emby, matched
// without regard to case, as the same request without it, marked as an
// Emby app's (see package embyapp), as Emby serves its API there as well as
// at its root. The web app manifest gets a neutral one, named after the
// server.
func underEmbyBase(next http.Handler, serverName func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := cutEmby(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if strings.EqualFold(rest, embyManifest) {
			name := "Polyfin"
			if serverName != nil {
				name = serverName()
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": name, "short_name": name, "start_url": "index.html", "display": "standalone",
				"background_color": "#101010", "theme_color": "#101010", "icons": []any{},
			})
			return
		}
		stripped := r.WithContext(embyapp.With(r.Context()))
		stripped.URL = new(url.URL)
		*stripped.URL = *r.URL
		stripped.URL.Path = rest
		stripped.URL.RawPath = ""
		if raw, ok := cutEmby(r.URL.RawPath); ok {
			stripped.URL.RawPath = raw
		}
		next.ServeHTTP(w, stripped)
	})
}

// cutEmby returns p without the Emby prefix, and whether p was under it.
func cutEmby(p string) (string, bool) {
	if len(p) < len(embyapp.Prefix) || !strings.EqualFold(p[:len(embyapp.Prefix)], embyapp.Prefix) {
		return "", false
	}
	rest := p[len(embyapp.Prefix):]
	switch {
	case rest == "":
		return "/", true
	case rest[0] == '/':
		return rest, true
	}
	return "", false
}

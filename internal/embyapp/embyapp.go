// Package embyapp tells the requests of Emby's own apps, such as Emby for
// Android TV, from those of Jellyfin's. Emby's apps speak the same API, but
// refuse a server that does not name itself an Emby server, and read a few
// answers their own way (see package jellyfin's emby.go).
//
// An Emby app shows itself by the /emby path it puts before every route,
// the "Emby" scheme of its authorization, or its X-Emby-Client header.
// Its image and stream addresses carry none of these: they carry its
// access token, which Remember keeps once one of its requests showed it.
package embyapp

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

// Prefix is the path Emby serves its API under, as well as at its root.
const Prefix = "/emby"

type key struct{}

// With marks ctx as a request of an Emby app.
func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, key{}, true)
}

// In reports whether ctx is a request of an Emby app (see Detect).
func In(ctx context.Context) bool {
	marked, _ := ctx.Value(key{}).(bool)
	return marked
}

// tokens are the access tokens Emby apps used, kept while Polyfin runs, up
// to maxTokens: a server rarely sees so many Emby devices.
var tokens = struct {
	sync.Mutex
	seen map[string]struct{}
}{seen: map[string]struct{}{}}

const maxTokens = 10000

// Remember keeps token as an Emby app's.
func Remember(token string) {
	if token == "" {
		return
	}
	tokens.Lock()
	defer tokens.Unlock()
	if len(tokens.seen) < maxTokens {
		tokens.seen[token] = struct{}{}
	}
}

func remembered(token string) bool {
	tokens.Lock()
	defer tokens.Unlock()
	_, known := tokens.seen[token]
	return known
}

// Detect reports whether r comes from an Emby app: marked by its /emby
// path (see With), by its authorization or headers, or by a token an Emby
// app used before.
func Detect(r *http.Request) bool {
	if In(r.Context()) {
		return true
	}
	if r.Header.Get("X-Emby-Client") != "" || queryValue(r, "X-Emby-Client") != "" {
		return true
	}
	for _, header := range []string{"Authorization", "X-Emby-Authorization"} {
		scheme, _, _ := strings.Cut(strings.TrimSpace(r.Header.Get(header)), " ")
		if strings.EqualFold(scheme, "Emby") {
			return true
		}
	}
	token := Token(r)
	return token != "" && remembered(token)
}

// Token is the access token r carries in any of the forms Emby's and
// Jellyfin's apps send it, or "".
func Token(r *http.Request) string {
	for _, header := range []string{"Authorization", "X-Emby-Authorization"} {
		value := r.Header.Get(header)
		lower := strings.ToLower(value)
		at := strings.Index(lower, "token=")
		if at < 0 {
			continue
		}
		rest := strings.TrimLeft(value[at+len("token="):], " ")
		if strings.HasPrefix(rest, `"`) {
			rest = rest[1:]
			if end := strings.IndexByte(rest, '"'); end >= 0 {
				return rest[:end]
			}
			return rest
		}
		token, _, _ := strings.Cut(rest, ",")
		return strings.TrimSpace(token)
	}
	for _, value := range []string{
		r.Header.Get("X-Emby-Token"),
		r.Header.Get("X-MediaBrowser-Token"),
		queryValue(r, "api_key"),
		queryValue(r, "ApiKey"),
		queryValue(r, "X-Emby-Token"),
	} {
		if value != "" {
			return value
		}
	}
	return ""
}

// queryValue is the query parameter name, matched without regard to case,
// as Jellyfin matches them.
func queryValue(r *http.Request, name string) string {
	for key, values := range r.URL.Query() {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

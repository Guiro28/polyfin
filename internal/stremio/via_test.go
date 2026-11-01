package stremio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Requests sent Via a proxy, and those of the addons ProxyAddons names a
// proxy for, go through it; the others, and confined ones, do not.
func TestRequestsGoThroughTheProxyTheyAreRoutedTo(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.URL.String())
		mu.Unlock()
		if r.URL.Path == "/manifest.json" {
			_, _ = io.WriteString(w, `{"id":"a","name":"A","version":"1","resources":["meta"],"types":["movie"]}`)
			return
		}
		_, _ = io.WriteString(w, "via")
	}))
	defer proxy.Close()
	addon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"b","name":"B","version":"1","resources":["meta"],"types":["movie"]}`)
	}))
	defer addon.Close()
	c := NewClient("test")

	response, err := c.Via(proxy.URL).Open(t.Context(), http.MethodGet, "http://source.example/movie.mkv", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if string(body) != "via" || len(asked) != 1 || asked[0] != "http://source.example/movie.mkv" {
		t.Fatalf("through the proxy: %q, the proxy was asked %v", body, asked)
	}

	proxiedManifest := "http://proxied.example/manifest.json"
	c.ProxyAddons(func(manifestURL string) string {
		if manifestURL == proxiedManifest {
			return proxy.URL
		}
		return ""
	})
	if manifest, err := c.Manifest(t.Context(), proxiedManifest, false); err != nil || manifest.ID != "a" {
		t.Fatalf("a proxied addon's manifest: %+v, %v", manifest, err)
	}
	if manifest, err := c.Manifest(t.Context(), addon.URL+"/manifest.json", false); err != nil || manifest.ID != "b" {
		t.Fatalf("another addon's manifest: %+v, %v", manifest, err)
	}
	if len(asked) != 2 || asked[1] != proxiedManifest {
		t.Errorf("the proxy was asked %v", asked)
	}
}

package playback

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/library"
)

// An addon routed through the server has its videos relayed even when the
// player could fetch them; one routed through a proxy has them fetched
// through it. A user's own addon, confined, is never routed.
func TestAddonRoutesRelayTheirVideos(t *testing.T) {
	server, proxied, direct := accounts.ID{1}, accounts.ID{2}, accounts.ID{3}
	for _, tc := range []struct {
		name     string
		addon    accounts.ID
		confined bool
		redirect bool
		viaProxy bool
	}{
		{"at the source", direct, false, true, false},
		{"through the server", server, false, false, false},
		{"through the proxy", proxied, false, false, true},
		{"a user's addon", proxied, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			origin := &fakeSource{body: []byte("\x1a\x45\xdf\xa3")}
			proxy := &fakeSource{body: []byte("\x1a\x45\xdf\xa3")}
			s := newService(t, origin, "ffprobe", nil)
			s.Routing(func(addon accounts.ID) Route {
				switch addon {
				case server:
					return Route{Relay: true}
				case proxied:
					return Route{Relay: true, Via: proxy}
				}
				return Route{}
			})
			version := library.Version{ID: accounts.ID{9}, URL: "https://93.184.216.34/movie.mkv", Confined: tc.confined,
				Origin: library.Origin{Addon: tc.addon}}
			request := httptest.NewRequest(http.MethodGet, "/Videos/x/stream", nil)
			request.Header.Set("Range", "bytes=0-3")
			response := httptest.NewRecorder()
			if err := s.Serve(response, request, version, Delivery{}); err != nil {
				t.Fatal(err)
			}
			if tc.redirect {
				if response.Code != http.StatusFound {
					t.Fatalf("got %d, want a redirect", response.Code)
				}
				return
			}
			if response.Code != http.StatusPartialContent || response.Body.String() != "\x1a\x45\xdf\xa3" {
				t.Fatalf("relay: %d %q", response.Code, response.Body.String())
			}
			if got := len(proxy.targets) > 0; got != tc.viaProxy {
				t.Errorf("through the proxy: %v, want %v", got, tc.viaProxy)
			}
			if tc.viaProxy && len(origin.targets) > 0 {
				t.Errorf("the server's own requests went to the source: %v", origin.targets)
			}
		})
	}
}

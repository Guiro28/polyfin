package admin

import (
	"net/http"
	"testing"
)

// A server addon's videos reach the apps as its route says, which an
// administrator sets; the proxy they take is a setting, an address checked.
func TestAddonRoutes(t *testing.T) {
	t.Parallel()
	api := newTestAPI(t, 10)
	administrator := api.signedIn("administrator", true)
	status, body, _ := administrator.call(http.MethodPost, "/scopes/shared/addons", map[string]string{"manifestUrl": localAddon(t)})
	if status != http.StatusCreated || body["route"] != "source" {
		t.Fatalf("installing an addon: %d %v", status, body)
	}
	id, _ := body["id"].(string)
	for _, route := range []string{"proxy", "server", "source"} {
		if status, body, _ := administrator.call(http.MethodPatch, "/scopes/shared/addons/"+id, map[string]string{"route": route}); status != http.StatusOK || body["route"] != route {
			t.Errorf("route %s: %d %v", route, status, body)
		}
	}
	if status, body, _ := administrator.call(http.MethodPatch, "/scopes/shared/addons/"+id, map[string]string{"route": "warp"}); status != http.StatusBadRequest || body["error"] != "invalid_route" {
		t.Errorf("an unknown route: %d %v", status, body)
	}

	_, settings, _ := administrator.call(http.MethodGet, "/settings", nil)
	if settings["streamProxy"] != "" {
		t.Errorf("default proxy: %v", settings["streamProxy"])
	}
	settings["streamProxy"] = "127.0.0.1:40000"
	if status, body, _ := administrator.call(http.MethodPut, "/settings", settings); status != http.StatusBadRequest || body["error"] != "invalid_stream_proxy" {
		t.Errorf("a proxy without its scheme: %d %v", status, body)
	}
	settings["streamProxy"] = "socks5://127.0.0.1:40000"
	if status, body, _ := administrator.call(http.MethodPut, "/settings", settings); status != http.StatusOK || body["streamProxy"] != "socks5://127.0.0.1:40000" {
		t.Errorf("saving a proxy: %d %v", status, body)
	}
}

package admin

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/moodiness/polyfin/internal/accounts"
)

// The proxy of the addons routed through one (see addons.RouteProxy) is
// checked through Cloudflare's trace page, which tells the address a
// request came from and whether it came through WARP; WARP's local proxy
// is found by checking where it usually listens.

// traceURL is Cloudflare's trace page.
const traceURL = "https://www.cloudflare.com/cdn-cgi/trace"

// traceTimeout bounds a check, a proxy that does not answer by then
// counting as unreachable.
const traceTimeout = 6 * time.Second

// warpCandidates are where WARP's local proxy usually listens: warp-cli's
// proxy mode, then the WARP Docker images, by the names a Compose file
// gives their service, and the host of a container.
var warpCandidates = []string{
	"socks5://127.0.0.1:40000",
	"socks5://127.0.0.1:1080",
	"socks5://127.0.0.1:9091",
	"socks5://warp:1080",
	"socks5://warp:9091",
	"socks5://warp-socks:9091",
	"socks5://cloudflare-warp:1080",
	"socks5://host.docker.internal:40000",
}

// proxyCheckJSON is what a check found: whether the trace page answered,
// whether through WARP, and the address it saw.
type proxyCheckJSON struct {
	Reachable bool   `json:"reachable"`
	Warp      bool   `json:"warp"`
	IP        string `json:"ip"`
}

// trace asks the trace page through proxy, a socks5:// or http://
// address, or from the server's own address when proxy is empty.
func trace(ctx context.Context, proxy string) proxyCheckJSON {
	ctx, cancel := context.WithTimeout(ctx, traceTimeout)
	defer cancel()
	transport := &http.Transport{
		DialContext:       (&net.Dialer{Timeout: traceTimeout}).DialContext,
		DisableKeepAlives: true,
	}
	if proxy != "" {
		address, err := url.Parse(proxy)
		if err != nil {
			return proxyCheckJSON{}
		}
		transport.Proxy = http.ProxyURL(address)
	}
	defer transport.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, traceURL, nil)
	if err != nil {
		return proxyCheckJSON{}
	}
	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		return proxyCheckJSON{}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<10))
	if err != nil || response.StatusCode != http.StatusOK {
		return proxyCheckJSON{}
	}
	check := proxyCheckJSON{Reachable: true}
	for line := range strings.SplitSeq(string(body), "\n") {
		key, value, _ := strings.Cut(strings.TrimSpace(line), "=")
		switch key {
		case "ip":
			check.IP = value
		case "warp":
			check.Warp = value == "on" || value == "plus"
		}
	}
	return check
}

// checkStreamProxy checks the proxy the body names, typed in the settings
// and not saved yet, or the server's own address when it names none.
func (h *handler) checkStreamProxy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Address string `json:"address"`
	}
	if !decode(w, r, &body) {
		return
	}
	address := strings.TrimSpace(body.Address)
	if address != "" && !accounts.ValidStreamProxy(address) {
		writeError(w, http.StatusBadRequest, "invalid_stream_proxy")
		return
	}
	writeJSON(w, http.StatusOK, trace(r.Context(), address))
}

// detectWarp looks for WARP's local proxy among warpCandidates, all at
// once, and answers the first that goes through WARP, in their order,
// empty when none does, with what the server's own address is: a server
// whose whole connection goes through WARP needs no proxy.
func (h *handler) detectWarp(w http.ResponseWriter, r *http.Request) {
	checks := make([]proxyCheckJSON, len(warpCandidates))
	var server proxyCheckJSON
	var wait sync.WaitGroup
	for i, candidate := range warpCandidates {
		wait.Go(func() { checks[i] = trace(r.Context(), candidate) })
	}
	wait.Go(func() { server = trace(r.Context(), "") })
	wait.Wait()
	found := ""
	for i, check := range checks {
		if check.Warp {
			found = warpCandidates[i]
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"address": found, "server": server})
}

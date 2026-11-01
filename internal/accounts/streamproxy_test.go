package accounts

import "testing"

func TestStreamProxiesAreProxyAddresses(t *testing.T) {
	for address, want := range map[string]bool{
		"socks5://127.0.0.1:40000":          true,
		"socks5h://warp:1080":               true,
		"http://user:secret@proxy.lan:8080": true,
		"https://proxy.example:443":         true,
		"127.0.0.1:40000":                   false,
		"socks5://127.0.0.1":                false,
		"ftp://proxy:21":                    false,
		"http://proxy:8080/path":            false,
		"http://proxy:8080?x=1":             false,
	} {
		if got := ValidStreamProxy(address); got != want {
			t.Errorf("%q: %v, want %v", address, got, want)
		}
	}
}

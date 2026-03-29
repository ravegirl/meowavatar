// Package httpclient provides a shared HTTP client with optional proxy support.
// Set PROXY_URL in the environment to route unofficial API requests through it.
// e.g. PROXY_URL=http://user:pass@host:port
package httpclient

import (
	"net/http"
	"net/url"
	"os"
	"sync"
)

var (
	proxied     *http.Client
	proxiedOnce sync.Once
)

// Proxied returns an HTTP client that routes through PROXY_URL if set.
// Initialized lazily so godotenv.Load() in main() runs first.
func Proxied() *http.Client {
	proxiedOnce.Do(func() {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if proxyURL := os.Getenv("PROXY_URL"); proxyURL != "" {
			if parsed, err := url.Parse(proxyURL); err == nil {
				transport.Proxy = http.ProxyURL(parsed)
			}
		}
		proxied = &http.Client{Transport: transport}
	})
	return proxied
}

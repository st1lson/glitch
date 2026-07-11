package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewProxyHandler creates an HTTP handler that reverse proxies requests
// to the target URL.
func NewProxyHandler(targetURL string) (http.Handler, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// Keep a reference to the original director
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Set the Host header to match the target URL,
		// otherwise many external services will reject the request with a 503 or 404.
		req.Host = parsedURL.Host
	}

	// Glitch is a development tool. Users frequently proxy to local or dev environments
	// that have self-signed certificates. We skip TLS verification to make this seamless.
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return proxy, nil
}

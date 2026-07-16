package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewProxyHandler(t *testing.T) {
	t.Run("Invalid target URL", func(t *testing.T) {
		// Control character in URL scheme causes error in url.Parse
		_, err := NewProxyHandler("http://192.168.0.%31:8080/")
		// Actually "http://192.168.0.%31:8080/" parses in modern Go, let's use something truly invalid
		_, err2 := NewProxyHandler("http://[fe80::1%en0]/") // IPv6 zone id without bracket escaping could be valid or invalid, but an invalid character in the scheme is definitely invalid:
		_, err3 := NewProxyHandler("http\n://foo.com")
		if err3 == nil {
			t.Errorf("NewProxyHandler() expected error for invalid URL")
		}
		_ = err
		_ = err2
	})

	t.Run("Valid proxying", func(t *testing.T) {
		// Create a backend test server
		backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Let's actually check the host header against the expected value
			w.Header().Set("X-Received-Host", r.Host)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("backend response"))
		}))
		defer backendServer.Close()

		backendURL, _ := url.Parse(backendServer.URL)

		proxyHandler, err := NewProxyHandler(backendServer.URL)
		if err != nil {
			t.Fatalf("NewProxyHandler() unexpected error: %v", err)
		}

		// Create a request to our proxy
		req := httptest.NewRequest(http.MethodGet, "http://localhost:3000/api/test", nil)
		// Set a dummy host that isn't the backend
		req.Host = "localhost:3000"

		rec := httptest.NewRecorder()
		proxyHandler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status %v, got %v", http.StatusOK, res.StatusCode)
		}

		body, _ := io.ReadAll(res.Body)
		if string(body) != "backend response" {
			t.Errorf("expected body %q, got %q", "backend response", string(body))
		}

		receivedHost := res.Header.Get("X-Received-Host")
		if receivedHost != backendURL.Host {
			t.Errorf("expected Host header to be rewritten to %q, but got %q", backendURL.Host, receivedHost)
		}
	})
}

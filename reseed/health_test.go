package reseed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHealth_ReturnsOK checks the liveness endpoint answers 200 with no UA or token.
func TestHealth_ReturnsOK(t *testing.T) {
	srv := NewServer("", false, "", 4, 40, 2000)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("Expected body ok, got %q", rec.Body.String())
	}
	if rec.Header().Get("Version") == "" {
		t.Fatal("Expected non-empty Version header")
	}
}

// TestHealth_BypassesRateLimiters exhausts the web bucket then proves /health still answers.
func TestHealth_BypassesRateLimiters(t *testing.T) {
	srv := NewServer("", false, "", 4, 40, 2000)
	// Drain the per-IP web bucket through / with the router UA (avoids homepage disk reads).
	seen429 := false
	for i := 0; i < 60; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", I2pUserAgent)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			seen429 = true
		}
	}
	if !seen429 {
		t.Fatal("Expected / to rate-limit to 429, it never did")
	}
	// The monitor path must be unaffected by the drained bucket, even from the same IP.
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Request %d: expected /health 200 under exhausted bucket, got %d", i, rec.Code)
		}
	}
}

// TestHealth_RespectsPrefix checks a prefixed server serves health under the prefix.
func TestHealth_RespectsPrefix(t *testing.T) {
	srv := NewServer("/netdb", false, "", 4, 40, 2000)
	req := httptest.NewRequest(http.MethodGet, "/netdb/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected prefixed /health 200, got %d", rec.Code)
	}
}

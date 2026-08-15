package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newRequestWithOrigin(origin string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/console", nil)
	req.Host = "panel.example.com"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

// A panel served from the same host as the API is not cross-site, so it must
// keep working even when CORS_ALLOWED_ORIGINS was never configured. Without
// this, shipping the origin check would silently kill the live console on
// exactly that (very common) deployment.
func TestCheckOrigin_SameHostAllowedWithoutConfiguration(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	req := newRequestWithOrigin("https://panel.example.com")
	if !upgrader.CheckOrigin(req) {
		t.Error("expected a same-origin request to be allowed with no CORS list configured")
	}

	req = newRequestWithOrigin("https://evil.example")
	if upgrader.CheckOrigin(req) {
		t.Error("expected a foreign origin to still be rejected with no CORS list configured")
	}
}

func TestSameOrigin(t *testing.T) {
	if !sameOrigin("https://panel.example.com", "panel.example.com") {
		t.Error("expected matching host to be same-origin")
	}
	if !sameOrigin("http://localhost:8080", "localhost:8080") {
		t.Error("expected matching host:port to be same-origin")
	}
	if sameOrigin("https://evil.example", "panel.example.com") {
		t.Error("expected a different host not to be same-origin")
	}
	// A port mismatch is a different origin, and it is also the shape a
	// same-site-but-different-app attack would take.
	if sameOrigin("http://localhost:5173", "localhost:8080") {
		t.Error("expected a port mismatch not to be same-origin")
	}
	if sameOrigin("", "panel.example.com") || sameOrigin("https://panel.example.com", "") {
		t.Error("expected empty origin or host never to be same-origin")
	}
}

// The console WebSocket is authenticated, so the risk CheckOrigin closes is
// CSWSH: a page the admin happens to visit opening a socket with their
// credentials and running server commands. These assert the allow-list, and
// that non-browser callers (no Origin header at all) still get through.

func TestOriginAllowed_DefaultsToLocalDevOrigins(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	for _, allowed := range []string{"http://localhost:5173", "http://localhost:8080"} {
		if !originAllowed(allowed) {
			t.Errorf("expected %q to be allowed by the default list", allowed)
		}
	}
	for _, denied := range []string{"https://evil.example", "http://localhost:9999", ""} {
		if originAllowed(denied) {
			t.Errorf("expected %q to be denied by the default list", denied)
		}
	}
}

func TestOriginAllowed_UsesConfiguredList(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://mine.example.com, https://panel.example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	if !originAllowed("https://mine.example.com") {
		t.Error("expected the first configured origin to be allowed")
	}
	// Whitespace around a comma-separated entry must not break the match.
	if !originAllowed("https://panel.example.com") {
		t.Error("expected the space-padded configured origin to be allowed")
	}
	if originAllowed("https://evil.example") {
		t.Error("expected an unlisted origin to be denied")
	}
	// Once a list is configured it fully replaces the localhost default.
	if originAllowed("http://localhost:5173") {
		t.Error("expected localhost to be denied once an explicit list is configured")
	}
}

func TestUpgraderCheckOrigin_AllowsOriginlessClients(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://mine.example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	req := newRequestWithOrigin("")
	if !upgrader.CheckOrigin(req) {
		t.Error("expected a client sending no Origin (bot/curl) to be allowed; auth still gates it")
	}

	req = newRequestWithOrigin("https://evil.example")
	if upgrader.CheckOrigin(req) {
		t.Error("expected a browser sending a foreign Origin to be rejected (CSWSH)")
	}

	req = newRequestWithOrigin("https://mine.example.com")
	if !upgrader.CheckOrigin(req) {
		t.Error("expected the configured origin to be accepted")
	}
}

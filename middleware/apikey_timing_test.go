package middleware

import (
	"os"
	"testing"
)

func TestAPIKeyValid_MatchesAndRejects(t *testing.T) {
	os.Setenv("API_KEY", "correct-horse-battery-staple")
	defer os.Unsetenv("API_KEY")

	if !apiKeyValid("correct-horse-battery-staple") {
		t.Error("expected the exact key to validate")
	}
	if apiKeyValid("correct-horse-battery-stapl") {
		t.Error("expected a truncated key to be rejected")
	}
	if apiKeyValid("wrong") {
		t.Error("expected a wrong key to be rejected")
	}
	if apiKeyValid("") {
		t.Error("expected an empty presented key to be rejected")
	}
}

// A server started without API_KEY must fail closed. Without the explicit
// empty check, an unset API_KEY would make ConstantTimeCompare("", "") == 1
// and every caller presenting an empty key would authenticate.
func TestAPIKeyValid_UnsetKeyFailsClosed(t *testing.T) {
	os.Unsetenv("API_KEY")

	if apiKeyValid("") {
		t.Error("expected an empty presented key to be rejected when API_KEY is unset")
	}
	if apiKeyValid("anything") {
		t.Error("expected any key to be rejected when API_KEY is unset")
	}
}

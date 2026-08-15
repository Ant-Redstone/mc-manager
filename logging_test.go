package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"log/slog"

	"github.com/gin-gonic/gin"
)

// The access log is the highest-volume thing this process writes and the one
// Loki queries hardest, so its shape is worth pinning: every line JSON, the
// level derived from the status, and no credential in it.

func captureLog(t *testing.T, level slog.Level, fn func()) []map[string]any {
	t.Helper()
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	fn()

	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q (%v)", line, err)
		}
		out = append(out, m)
	}
	return out
}

func requestThrough(t *testing.T, status int, target string) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestLogger())
	r.GET("/*any", func(c *gin.Context) { c.Status(status) })

	lines := captureLog(t, slog.LevelInfo, func() {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	})
	if len(lines) != 1 {
		t.Fatalf("expected exactly one access-log line, got %d: %v", len(lines), lines)
	}
	return lines[0]
}

func TestAccessLog_StatusPicksTheLevel(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   string
	}{
		{200, "INFO"},
		{204, "INFO"},
		{304, "INFO"},
		{400, "WARN"},
		{403, "WARN"}, // a wave of these is what a missing role looks like
		{404, "WARN"},
		{500, "ERROR"},
		{503, "ERROR"},
	} {
		line := requestThrough(t, tc.status, "/api/thing")
		if line["level"] != tc.want {
			t.Errorf("status %d: expected level %s, got %v", tc.status, tc.want, line["level"])
		}
		if line["msg"] != "http request" {
			t.Errorf("status %d: unexpected msg %v", tc.status, line["msg"])
		}
	}
}

func TestAccessLog_CarriesTheFieldsWorthQuerying(t *testing.T) {
	line := requestThrough(t, http.StatusOK, "/api/players")
	for _, key := range []string{"status", "method", "path", "latency_ms", "ip", "time", "level", "msg"} {
		if _, ok := line[key]; !ok {
			t.Errorf("access log is missing %q: %v", key, line)
		}
	}
}

func TestAccessLog_RedactsCredentialsInTheQuery(t *testing.T) {
	line := requestThrough(t, http.StatusOK, "/api/console?token=eyJhbGci.SUPERSECRET.sig&key=alsosecret")

	blob, _ := json.Marshal(line)
	for _, secret := range []string{"SUPERSECRET", "alsosecret"} {
		if strings.Contains(string(blob), secret) {
			t.Errorf("secret %q reached the access log: %s", secret, blob)
		}
	}
	if got, _ := line["path"].(string); !strings.Contains(got, "token=REDACTED") || !strings.Contains(got, "key=REDACTED") {
		t.Errorf("expected both credentials redacted, got path=%q", got)
	}
}

func TestLogLevel_DefaultsToInfoAndIgnoresGarbage(t *testing.T) {
	for env, want := range map[string]slog.Level{
		"":        slog.LevelInfo,
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		" warn ":  slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		// A typo must never be the reason the server won't start or goes silent.
		"lowd": slog.LevelInfo,
	} {
		t.Setenv("LOG_LEVEL", env)
		if got := logLevel(); got != want {
			t.Errorf("LOG_LEVEL=%q: expected %v, got %v", env, want, got)
		}
	}
}

func TestLogHandler_JSONInProductionTextForDevelopers(t *testing.T) {
	t.Setenv("GIN_MODE", "")
	if _, ok := newLogHandler(os.Stdout).(*slog.JSONHandler); !ok {
		t.Error("expected a JSON handler when GIN_MODE is not debug -- Promtail ships this")
	}
	t.Setenv("GIN_MODE", "debug")
	if _, ok := newLogHandler(os.Stdout).(*slog.TextHandler); !ok {
		t.Error("expected a text handler for a developer running with GIN_MODE=debug")
	}
}

package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
)

// setupLogging installs the process-wide structured logger.
//
// The homelab ships container logs to Loki via Promtail (see
// docker-compose.yml). Loki can filter on a JSON field but not on a sentence,
// so `log.Printf("failed to list backups: %v", err)` -- which is what this
// codebase used everywhere -- is unqueryable: there is no level to alert on
// and no field to group by. Everything below exists to make `{job="mc-manager"}
// | json | level="ERROR"` a thing you can actually write.
//
// slog.SetDefault also redirects the standard `log` package through this
// handler, so anything not yet converted (and anything a dependency logs)
// still comes out as structured JSON rather than falling back to plain text
// halfway down the file.
func setupLogging() {
	handler := newLogHandler(os.Stdout)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Drop the standard logger's own "2009/11/10 23:00:00 " prefix: slog
	// already stamps a time field, and leaving both makes the JSON message
	// start with a duplicate timestamp.
	log.SetFlags(0)
}

// fatal logs a boot failure at ERROR and exits non-zero -- log.Fatalf's
// behaviour, but through the structured handler so a failed boot is as
// queryable as anything else. Without this, the one class of message that
// matters most (the server refusing to start) would be the only one Loki
// couldn't filter on level.
func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}

func newLogHandler(w *os.File) slog.Handler {
	opts := &slog.HandlerOptions{Level: logLevel()}

	// Human-readable text when someone is watching a terminal, JSON when a log
	// shipper is. GIN_MODE=debug is the switch the project already uses for
	// "this is a developer running it", so it decides this too rather than
	// introducing a second, competing notion of environment.
	if strings.EqualFold(os.Getenv("GIN_MODE"), "debug") {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

// logLevel reads LOG_LEVEL (debug|info|warn|error), defaulting to info. An
// unrecognised value falls back to info rather than failing to boot: a typo in
// an env var must never be the reason the server won't start.
func logLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

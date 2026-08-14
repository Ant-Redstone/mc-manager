package services

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	present := dir + "/present.txt"
	os.WriteFile(present, []byte("x"), 0644)

	if !fileExists(present) {
		t.Error("expected fileExists to be true for an existing file")
	}
	if fileExists(dir + "/missing.txt") {
		t.Error("expected fileExists to be false for a missing file")
	}
}

// rotated() reads from the fixed LatestLogPath constant internally, so these
// tests set up real files there (via setupServerDir/writeServerFile) rather
// than passing in an arbitrary path.

func TestRotated_SameFileNotTruncated(t *testing.T) {
	setupServerDir(t)
	path := writeServerFile(t, "logs/latest.log", "hello\n")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if rotated(f, info) {
		t.Error("expected rotated to be false for an untouched file")
	}
}

func TestRotated_DifferentFileDetected(t *testing.T) {
	setupServerDir(t)
	path := writeServerFile(t, "logs/latest.log", "hello\n")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Simulate Minecraft rotating the log: latest.log is removed and a new
	// file takes its place at the same path.
	if err := os.Remove(path); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}
	writeServerFile(t, "logs/latest.log", "a fresh session\n")

	if !rotated(f, info) {
		t.Error("expected rotated to be true when a new file now occupies the path")
	}
}

func TestRotated_TruncatedInPlace(t *testing.T) {
	setupServerDir(t)
	path := writeServerFile(t, "logs/latest.log", "hello world, this is a long line\n")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	// Advance the read offset past where the file will be truncated to.
	f.Seek(20, 0)

	if err := os.Truncate(path, 5); err != nil {
		t.Fatalf("failed to truncate file: %v", err)
	}

	if !rotated(f, info) {
		t.Error("expected rotated to be true after in-place truncation")
	}
}

func TestRotated_MissingPathIsNotRotated(t *testing.T) {
	setupServerDir(t)
	path := writeServerFile(t, "logs/latest.log", "hello\n")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}

	// A briefly-missing path mid-rotation should be tolerated, not reported
	// as rotated, so the tailer waits rather than thrashing.
	if rotated(f, info) {
		t.Error("expected rotated to be false when the path is briefly missing")
	}
}

func TestGetLogHub_NonNilAfterStart(t *testing.T) {
	setupServerDir(t)
	StartLogTailer()
	if GetLogHub() == nil {
		t.Error("expected GetLogHub to be non-nil after StartLogTailer")
	}
}

// openBacklogFile writes content to a temp file and opens it for readBacklog
// tests — readBacklog operates on an already-open *os.File, so these don't
// need the fixed LatestLogPath.
func openBacklogFile(t *testing.T, content string) *os.File {
	t.Helper()
	path := t.TempDir() + "/latest.log"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func restOfFile(t *testing.T, f *os.File) string {
	t.Helper()
	rest, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read rest of file: %v", err)
	}
	return string(rest)
}

func TestReadBacklog_CompleteLines(t *testing.T) {
	f := openBacklogFile(t, "line one\nline two\nline three\n")

	lines := readBacklog(f)

	if len(lines) != 3 || lines[0] != "line one" || lines[2] != "line three" {
		t.Errorf("expected the 3 complete lines, got %q", lines)
	}
	if rest := restOfFile(t, f); rest != "" {
		t.Errorf("expected the offset at EOF after a newline-terminated file, rest=%q", rest)
	}
}

func TestReadBacklog_TrailingPartialLeftForTheTailLoop(t *testing.T) {
	f := openBacklogFile(t, "done line\nstill being writ")

	lines := readBacklog(f)

	if len(lines) != 1 || lines[0] != "done line" {
		t.Errorf("expected only the complete line, got %q", lines)
	}
	// The partial must be readable, whole, from where readBacklog left the
	// offset — that's what hands it to the tail loop unsplit.
	if rest := restOfFile(t, f); rest != "still being writ" {
		t.Errorf("expected the offset at the start of the partial line, rest=%q", rest)
	}
}

func TestReadBacklog_CapsAtBacklogMaxLines(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= backlogMaxLines+50; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	f := openBacklogFile(t, b.String())

	lines := readBacklog(f)

	if len(lines) != backlogMaxLines {
		t.Fatalf("expected exactly %d lines, got %d", backlogMaxLines, len(lines))
	}
	if lines[len(lines)-1] != fmt.Sprintf("line %d", backlogMaxLines+50) {
		t.Errorf("expected the newest line last, got %q", lines[len(lines)-1])
	}
}

func TestReadBacklog_LargeFileReadsOnlyTheWindow(t *testing.T) {
	// A file larger than the read window: the seed must still return the
	// newest lines, and the first (likely mid-line) window fragment must be
	// dropped rather than emitted as a garbage half-line.
	longLine := strings.Repeat("x", 1000)
	var b strings.Builder
	for i := 1; i <= 400; i++ { // 400 KB > backlogWindowBytes
		fmt.Fprintf(&b, "%s %d\n", longLine, i)
	}
	f := openBacklogFile(t, b.String())

	lines := readBacklog(f)

	if len(lines) != backlogMaxLines {
		t.Fatalf("expected %d lines, got %d", backlogMaxLines, len(lines))
	}
	if !strings.HasSuffix(lines[len(lines)-1], " 400") {
		t.Errorf("expected the newest line last, got %q", lines[len(lines)-1])
	}
	for i, l := range lines {
		if !strings.HasPrefix(l, "x") || len(l) < 1000 {
			t.Errorf("line %d looks like a window-cut fragment: %q…", i, l[:min(len(l), 40)])
		}
	}
}

func TestReadBacklog_EmptyFile(t *testing.T) {
	f := openBacklogFile(t, "")

	if lines := readBacklog(f); lines != nil {
		t.Errorf("expected nil for an empty file, got %q", lines)
	}
}

func TestReadBacklog_OnlyAPartialFirstLine(t *testing.T) {
	f := openBacklogFile(t, "no newline yet")

	lines := readBacklog(f)

	if lines != nil {
		t.Errorf("expected nothing to seed, got %q", lines)
	}
	// The whole (partial) first line stays with the tail loop.
	if rest := restOfFile(t, f); rest != "no newline yet" {
		t.Errorf("expected the offset rewound to the file start, rest=%q", rest)
	}
}

func TestReadBacklog_StripsCarriageReturns(t *testing.T) {
	f := openBacklogFile(t, "windows line\r\nanother\r\n")

	lines := readBacklog(f)

	if len(lines) != 2 || lines[0] != "windows line" || lines[1] != "another" {
		t.Errorf("expected CRs stripped, got %q", lines)
	}
}

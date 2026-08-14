package services

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lomokwa/mc-manager/types"
)

const logPollInterval = 200 * time.Millisecond

const (
	// backlogMaxLines matches the hub's replay buffer size — seeding more
	// than the hub can hold would just be dropped again on subscribe.
	backlogMaxLines = 200
	// backlogWindowBytes bounds how much of a large log is read to find those
	// lines. 256 KiB comfortably covers 200 Minecraft log lines (~100 bytes
	// each is typical; even pathological stack traces fit).
	backlogWindowBytes = 256 * 1024
)

var (
	tailHub  *types.LogHub
	tailOnce sync.Once
)

// StartLogTailer creates the process-wide log hub and begins following
// LatestLogPath. Call once at API startup. Unlike the old in-process
// exec.Cmd's stdout pump, this hub is never closed on server stop — the
// minecraft container (and its log file) can outlive any single API process,
// so the hub's lifetime now matches the API's, not the JVM's.
func StartLogTailer() {
	tailOnce.Do(func() {
		tailHub = types.NewLogHub()
		go tailLoop()
	})
}

// GetLogHub returns the long-lived hub fed by the tailer. Non-nil once
// StartLogTailer has been called (which main.go does at boot), so callers no
// longer need to guard against a nil hub between server starts.
func GetLogHub() *types.LogHub {
	return tailHub
}

// tailLoop follows LatestLogPath, broadcasting each complete line to tailHub.
// Minecraft rotates this file on every JVM start (a fresh file replaces the
// old one) — this has bitten the project before, so rotation is detected two
// ways rather than trusting file position alone: the file's identity
// (os.SameFile) or its size shrinking under our read offset (truncation).
// Either signals "reopen from the top", since the new session's own readiness
// line ("Done (...)") must not be missed.
func tailLoop() {
	var (
		f       *os.File
		reader  *bufio.Reader
		info    os.FileInfo
		partial []byte
	)

	// If the file already exists right now, we're attaching to a server that
	// may already be running — seed the hub's replay buffer with the tail of
	// its existing output (so a console opened right after an API restart is
	// never blank while the JVM sits mid-session), then continue tailing from
	// there. A file that doesn't exist yet is a fresh session: read it from
	// the start once it appears, so the "Done" line isn't missed.
	seedBacklogOnOpen := fileExists(LatestLogPath)

	openCurrent := func() bool {
		nf, err := os.Open(LatestLogPath)
		if err != nil {
			return false
		}
		fi, err := nf.Stat()
		if err != nil {
			nf.Close()
			return false
		}
		if seedBacklogOnOpen {
			for _, line := range readBacklog(nf) {
				tailHub.Broadcast(line)
			}
			seedBacklogOnOpen = false
		}
		f, reader, info, partial = nf, bufio.NewReader(nf), fi, nil
		return true
	}

	for {
		if f == nil {
			if !openCurrent() {
				time.Sleep(time.Second)
				continue
			}
		}

		for {
			chunk, err := reader.ReadBytes('\n')
			if len(chunk) > 0 {
				partial = append(partial, chunk...)
				if partial[len(partial)-1] == '\n' {
					tailHub.Broadcast(strings.TrimRight(string(partial), "\r\n"))
					partial = nil
				}
			}
			if err != nil {
				break // caught up (EOF) or a real read error — either way, stop draining
			}
		}

		if rotated(f, info) {
			f.Close()
			f = nil // reopened on the next loop iteration, from offset 0
			continue
		}
		time.Sleep(logPollInterval)
	}
}

// readBacklog returns up to backlogMaxLines complete lines from the end of f,
// reading at most backlogWindowBytes, and positions f so tailing resumes
// exactly where the returned lines stop: at EOF when the file ends in a
// newline, or at the start of the trailing partial line the JVM is still
// writing — that way the partial is later emitted once, whole, by the normal
// tail loop instead of being split across the seed and the tail.
func readBacklog(f *os.File) []string {
	fi, err := f.Stat()
	if err != nil {
		f.Seek(0, io.SeekEnd)
		return nil
	}

	start := fi.Size() - backlogWindowBytes
	windowTruncated := start > 0
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		f.Seek(0, io.SeekEnd)
		return nil
	}
	data, err := io.ReadAll(f) // leaves the offset at EOF
	if err != nil {
		f.Seek(0, io.SeekEnd)
		return nil
	}

	lastNL := bytes.LastIndexByte(data, '\n')
	if lastNL < 0 {
		// No complete line in the window. A small file whose first line is
		// still being written: rewind so the tail loop emits it whole later.
		// (With windowTruncated this would be one >256KiB line — degenerate;
		// leaving the offset at EOF just skips the unreadable fragment.)
		if !windowTruncated {
			f.Seek(start, io.SeekStart)
		}
		return nil
	}
	if lastNL < len(data)-1 {
		// Trailing partial line: hand it back to the tail loop.
		f.Seek(start+int64(lastNL)+1, io.SeekStart)
	}

	lines := strings.Split(string(data[:lastNL]), "\n")
	if windowTruncated && len(lines) > 0 {
		lines = lines[1:] // the window almost certainly opened mid-line
	}
	if len(lines) > backlogMaxLines {
		lines = lines[len(lines)-backlogMaxLines:]
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func rotated(f *os.File, openedInfo os.FileInfo) bool {
	pathInfo, err := os.Stat(LatestLogPath)
	if err != nil {
		return false // briefly missing mid-rotation — wait rather than thrash
	}
	if !os.SameFile(openedInfo, pathInfo) {
		return true // the path now points at a different file
	}
	curInfo, err := f.Stat()
	if err != nil {
		return true
	}
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return true
	}
	return curInfo.Size() < offset // truncated in place
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

//go:build linux

// mc-supervisor runs as PID 1 (under tini) inside the "minecraft" container.
// It owns the Minecraft JVM as its direct child process and exposes a control
// plane on the shared server-directory volume so the "mc-manager" API
// container — which no longer has a handle on the JVM — can start/stop it and
// send it console commands without a network channel between the two:
//
//   - console.in (FIFO)  — raw console commands, forwarded verbatim to the
//     JVM's stdin.
//   - control.in (FIFO)  — lifecycle verbs: START, STOP, RESTART, KILL.
//   - status.json (file) — atomically-replaced heartbeat the API reads to
//     answer "is it running" (running/pid/since/heartbeat/desired).
//
// The JVM's own stdout is left alone — Minecraft already mirrors it to
// logs/latest.log on disk, which the API tails directly (see
// services/logtail.go). This process's only relationship to that log file is
// that it doesn't touch it.
package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lomokwa/mc-manager/types"
)

var (
	serverDir   = env("MC_SERVER_DIR", "/mc")
	controlDir  = filepath.Join(serverDir, ".mcmanager")
	consoleFifo = filepath.Join(controlDir, "console.in")
	controlFifo = filepath.Join(controlDir, "control.in")
	statusFile  = filepath.Join(controlDir, "status.json")
	tmpStatus   = statusFile + ".tmp"
	desiredFile = filepath.Join(controlDir, "desired")

	serverJar   = env("MC_SERVER_JAR", "server.jar")
	javaXms     = env("MC_JAVA_XMS", "1G")
	javaXmx     = env("MC_JAVA_XMX", "2G")
	stopTimeout = envDuration("MC_STOP_TIMEOUT", 30*time.Second)
	heartbeat   = envDuration("MC_HEARTBEAT", 2*time.Second)
)

// supervisor guards the single in-flight JVM child process. Only one exists
// per container; all fields are protected by mu.
type supervisor struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	stdin io.WriteCloser
	since time.Time
	done  chan struct{} // closed when the current JVM has fully exited
}

// fatal logs at ERROR and exits non-zero -- log.Fatalf's behaviour, through
// the structured handler so a supervisor that refuses to start is as queryable
// in Loki as anything else it emits.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	// JSON on stdout, which docker captures and Promtail ships. The "service"
	// attribute is what separates these lines from the JVM's own output in the
	// same container log.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "mc-supervisor"))

	if err := os.MkdirAll(controlDir, 0o770); err != nil {
		fatal("mkdir control dir", "dir", controlDir, "err", err)
	}
	for _, p := range []string{consoleFifo, controlFifo} {
		if err := ensureFifo(p); err != nil {
			fatal("mkfifo", "path", p, "err", err)
		}
	}

	s := &supervisor{}
	s.writeStatus() // publish running=false immediately so the API sees a live heartbeat

	go s.forwardConsole()
	go s.readControl()
	go func() {
		t := time.NewTicker(heartbeat)
		defer t.Stop()
		for range t.C {
			s.writeStatus()
		}
	}()

	// docker stop / host shutdown: save and stop gracefully, same as an
	// explicit STOP, before this process exits.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		slog.Info("termination signal received, stopping the JVM gracefully")
		s.stop()
		s.writeStatus()
		os.Exit(0)
	}()

	// Reboot/redeploy recovery: if the server was running when this container
	// last stopped, bring it back without a human needing to click Start again.
	if readDesired() == "running" {
		slog.Info("desired state is running, auto-starting the JVM")
		if _, err := s.start(); err != nil {
			slog.Error("auto-start failed", "err", err)
		}
		s.writeStatus()
	}

	select {} // the goroutines above do all the work
}

func (s *supervisor) start() (started bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil {
		return false, nil // already running — starting is idempotent
	}

	cmd := exec.Command("java", "-Xms"+javaXms, "-Xmx"+javaXmx, "-jar", serverJar, "nogui")
	cmd.Dir = serverDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false, err
	}
	// Visible via `docker logs minecraft` for operators; the API reads the
	// authoritative stream from logs/latest.log instead (see logtail.go).
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return false, err
	}

	done := make(chan struct{})
	s.cmd, s.stdin, s.since, s.done = cmd, stdin, time.Now().UTC(), done

	go func() {
		waitErr := cmd.Wait()
		slog.Warn("JVM exited", "err", waitErr)
		s.mu.Lock()
		s.cmd, s.stdin, s.since = nil, nil, time.Time{}
		s.mu.Unlock()
		close(done)
		s.writeStatus()
	}()

	slog.Info("JVM started", "pid", cmd.Process.Pid)
	return true, nil
}

// stop sends the graceful "stop" console command and waits up to stopTimeout
// for the JVM to exit on its own before force-killing it. A no-op if nothing
// is running.
func (s *supervisor) stop() {
	s.mu.Lock()
	cmd, stdin, done := s.cmd, s.stdin, s.done
	s.mu.Unlock()
	if cmd == nil {
		return
	}
	if stdin != nil {
		io.WriteString(stdin, "stop\n")
	}
	select {
	case <-done:
	case <-time.After(stopTimeout):
		slog.Warn("JVM did not stop in time, killing", "timeout", stopTimeout)
		cmd.Process.Kill()
		<-done
	}
}

// forwardConsole relays every line written to the console FIFO into the JVM's
// stdin. Opened O_RDWR (not O_RDONLY) so this end never sees EOF when a
// writer briefly disconnects — the pipe simply idles until the next writer.
func (s *supervisor) forwardConsole() {
	f, err := os.OpenFile(consoleFifo, os.O_RDWR, 0)
	if err != nil {
		fatal("open console fifo", "err", err)
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			s.mu.Lock()
			w := s.stdin
			s.mu.Unlock()
			if w != nil {
				io.WriteString(w, line)
			}
			// If the JVM isn't running there's nowhere for the command to go;
			// drop it. The API already refuses to send when !IsServerRunning().
		}
		if err != nil {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// readControl handles lifecycle verbs. Kept on a separate FIFO from console
// commands: "stop" is a legitimate game command, and START has no stdin to
// write to when the JVM is down, so lifecycle can't share the console channel.
func (s *supervisor) readControl() {
	f, err := os.OpenFile(controlFifo, os.O_RDWR, 0)
	if err != nil {
		fatal("open control fifo", "err", err)
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		switch strings.ToUpper(strings.TrimSpace(line)) {
		case "START":
			writeDesired("running")
			if _, e := s.start(); e != nil {
				slog.Error("control verb start failed", "err", e)
			}
			s.writeStatus()
		case "STOP":
			writeDesired("stopped")
			s.stop()
			s.writeStatus()
		case "RESTART":
			writeDesired("running")
			s.stop()
			if _, e := s.start(); e != nil {
				slog.Error("control verb restart failed", "err", e)
			}
			s.writeStatus()
		case "KILL":
			writeDesired("stopped")
			s.mu.Lock()
			cmd := s.cmd
			s.mu.Unlock()
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
			}
		case "":
			// blank line between commands — ignore
		default:
			slog.Warn("unknown control verb", "verb", strings.TrimSpace(line))
		}
		if err != nil {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (s *supervisor) writeStatus() {
	s.mu.Lock()
	st := types.ServerRuntimeStatus{Heartbeat: time.Now().UTC(), Desired: readDesired()}
	if s.cmd != nil && s.cmd.Process != nil {
		st.Running, st.PID, st.Since = true, s.cmd.Process.Pid, s.since
	}
	s.mu.Unlock()

	b, err := json.Marshal(st)
	if err != nil {
		slog.Error("marshal status", "err", err)
		return
	}
	if err := os.WriteFile(tmpStatus, b, 0o644); err != nil {
		slog.Error("write status", "err", err)
		return
	}
	if err := os.Rename(tmpStatus, statusFile); err != nil {
		slog.Error("publish status", "err", err)
	}
}

// ensureFifo makes sure path exists as a named pipe, replacing anything else
// found there (e.g. a stray regular file left over from a bad state).
func ensureFifo(path string) error {
	if fi, err := os.Stat(path); err == nil {
		if fi.Mode()&os.ModeNamedPipe != 0 {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return syscall.Mkfifo(path, 0o660)
}

func readDesired() string {
	b, err := os.ReadFile(desiredFile)
	if err == nil && strings.TrimSpace(string(b)) == "running" {
		return "running"
	}
	return "stopped"
}

func writeDesired(d string) {
	if err := os.WriteFile(desiredFile, []byte(d), 0o644); err != nil {
		slog.Error("write desired state", "err", err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

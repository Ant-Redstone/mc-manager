package services

import (
	"testing"

	"github.com/lomokwa/mc-manager/db"
)

// TestDefaultRuntime_MatchesExistingConstants is the regression guard for
// Phase 1's central promise: introducing the registry does not move
// anything. Every path DefaultRuntime() derives must equal its constants.go
// counterpart byte-for-byte, not merely "resolve to the same place" -- see
// runtime.go's doc comment on why plain string concatenation (not
// filepath.Join) is what makes that possible.
func TestDefaultRuntime_MatchesExistingConstants(t *testing.T) {
	rt := DefaultRuntime()

	if rt.ID != DefaultServerID {
		t.Errorf("expected id %q, got %q", DefaultServerID, rt.ID)
	}
	if rt.Dir != ServerDir {
		t.Errorf("expected dir %q, got %q", ServerDir, rt.Dir)
	}
	if got := rt.ControlDir(); got != ControlDir {
		t.Errorf("ControlDir: expected %q, got %q", ControlDir, got)
	}
	if got := rt.ConsoleFifoPath(); got != ConsoleFifoPath {
		t.Errorf("ConsoleFifoPath: expected %q, got %q", ConsoleFifoPath, got)
	}
	if got := rt.ControlFifoPath(); got != ControlFifoPath {
		t.Errorf("ControlFifoPath: expected %q, got %q", ControlFifoPath, got)
	}
	if got := rt.StatusFilePath(); got != StatusFilePath {
		t.Errorf("StatusFilePath: expected %q, got %q", StatusFilePath, got)
	}
	if got := rt.LatestLogPath(); got != LatestLogPath {
		t.Errorf("LatestLogPath: expected %q, got %q", LatestLogPath, got)
	}
	if got := rt.ServerJarPath(); got != ServerJarPath {
		t.Errorf("ServerJarPath: expected %q, got %q", ServerJarPath, got)
	}
	if got := rt.ServerMetaPath(); got != ServerMetaPath {
		t.Errorf("ServerMetaPath: expected %q, got %q", ServerMetaPath, got)
	}
	// The one deliberate exception: BackupDir is NOT <dir>/backups for the
	// default server. The plan requires it stay exactly the pre-existing
	// BackupDir constant, since that's where the live backups already are.
	if got := rt.BackupDir(); got != BackupDir {
		t.Errorf("BackupDir: expected %q (unmoved), got %q", BackupDir, got)
	}
}

// TestServerRuntime_PathDerivation_NonDefault exercises a hypothetical
// second server, proving the derivation methods generalize beyond the
// default id/dir rather than accidentally hardcoding either.
func TestServerRuntime_PathDerivation_NonDefault(t *testing.T) {
	rt := &ServerRuntime{ID: "survival", Dir: "./servers/survival"}

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"ControlDir", rt.ControlDir(), "./servers/survival/.mcmanager"},
		{"ConsoleFifoPath", rt.ConsoleFifoPath(), "./servers/survival/.mcmanager/console.in"},
		{"ControlFifoPath", rt.ControlFifoPath(), "./servers/survival/.mcmanager/control.in"},
		{"StatusFilePath", rt.StatusFilePath(), "./servers/survival/.mcmanager/status.json"},
		{"LatestLogPath", rt.LatestLogPath(), "./servers/survival/logs/latest.log"},
		{"ServerJarPath", rt.ServerJarPath(), "./servers/survival/server.jar"},
		{"ServerMetaPath", rt.ServerMetaPath(), "./servers/survival/server-meta.json"},
		// Non-default servers DO get the id-namespaced backup path -- only
		// the default is special-cased to stay unmoved.
		{"BackupDir", rt.BackupDir(), "./backups/survival"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: expected %q, got %q", c.name, c.want, c.got)
		}
	}
}

// TestLoadRuntimes_StartsTailerForEachRegistryRow uses a second, uniquely
// named registry row (never touched by any other test in this package) so
// this test can prove LoadRuntimes actually built and started a tailer for
// it -- as opposed to observing a hub that some earlier test already warmed
// up on the shared "default" runtime.
func TestLoadRuntimes_StartsTailerForEachRegistryRow(t *testing.T) {
	setupTestDB(t)
	setupServerDir(t)

	if err := EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	extraDir := t.TempDir()
	if _, err := db.DB.Exec(
		`INSERT INTO servers (id, name, dir, port, jar, xms, xmx, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"rt-test-extra", "Extra", extraDir, 25566, "server.jar", "1G", "2G", 1,
	); err != nil {
		t.Fatalf("failed to insert extra server row: %v", err)
	}

	if err := LoadRuntimes(); err != nil {
		t.Fatalf("failed to load runtimes: %v", err)
	}

	runtimesMu.RLock()
	rt, ok := runtimes["rt-test-extra"]
	runtimesMu.RUnlock()
	if !ok {
		t.Fatal("expected LoadRuntimes to register a runtime for the extra server row")
	}
	if rt.Dir != extraDir {
		t.Errorf("expected dir %q, got %q", extraDir, rt.Dir)
	}
	if rt.Hub == nil {
		t.Error("expected LoadRuntimes to start a log tailer (non-nil hub) for the extra server")
	}

	// And the default row's own runtime/hub came from the very same call.
	defRT := DefaultRuntime()
	if defRT.Hub == nil {
		t.Error("expected LoadRuntimes to also start the default runtime's tailer")
	}
	if rt.Hub == defRT.Hub {
		t.Error("expected each server to get its own independent hub, not a shared one")
	}
}

func TestGetOrCreateRuntime_SameIDReturnsSamePointer(t *testing.T) {
	a := getOrCreateRuntime("shared-id-test", "./somewhere")
	b := getOrCreateRuntime("shared-id-test", "./somewhere-else-that-should-be-ignored")

	if a != b {
		t.Error("expected the same id to resolve to the same cached *ServerRuntime")
	}
	if a.Dir != "./somewhere" {
		t.Errorf("expected the first call's dir to win, got %q", a.Dir)
	}
}

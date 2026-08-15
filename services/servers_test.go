package services

import (
	"testing"

	"github.com/lomokwa/mc-manager/db"
)

func TestEnsureDefaultServer_SeedsOneRowPointingAtExistingServerDir(t *testing.T) {
	setupTestDB(t)

	if err := EnsureDefaultServer(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	servers, err := ListServers()
	if err != nil {
		t.Fatalf("failed to list servers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected exactly 1 server, got %d: %+v", len(servers), servers)
	}

	s := servers[0]
	if s.ID != DefaultServerID {
		t.Errorf("expected id %q, got %q", DefaultServerID, s.ID)
	}
	if s.Name != "Default" {
		t.Errorf("expected name %q, got %q", "Default", s.Name)
	}
	// The whole point of this row: it must describe the directory that
	// already exists today, not a new one -- see the ServerDir constant.
	if s.Dir != ServerDir {
		t.Errorf("expected dir %q (the existing ServerDir, unmoved), got %q", ServerDir, s.Dir)
	}
	if s.Port != 25565 {
		t.Errorf("expected port 25565, got %d", s.Port)
	}
	if s.VoicePort != nil {
		t.Errorf("expected voice_port to be NULL, got %v", *s.VoicePort)
	}
	if s.Jar != "server.jar" {
		t.Errorf("expected jar %q, got %q", "server.jar", s.Jar)
	}
	if s.Xms != "1G" || s.Xmx != "2G" {
		t.Errorf("expected xms/xmx to match the existing defaults (1G/2G), got %s/%s", s.Xms, s.Xmx)
	}
	if s.Sort != 0 {
		t.Errorf("expected sort 0, got %d", s.Sort)
	}
	if s.CreatedAt == "" {
		t.Error("expected created_at to be populated")
	}
}

func TestEnsureDefaultServer_IdempotentAndLeavesExistingRowUntouched(t *testing.T) {
	setupTestDB(t)

	if err := EnsureDefaultServer(); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Mutate the seeded row so a second call that (incorrectly) re-inserted
	// or overwrote it would be visible below.
	if _, err := db.DB.Exec(`UPDATE servers SET name = ? WHERE id = ?`, "Renamed", DefaultServerID); err != nil {
		t.Fatalf("failed to mutate seeded row: %v", err)
	}

	if err := EnsureDefaultServer(); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	servers, err := ListServers()
	if err != nil {
		t.Fatalf("failed to list servers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected still exactly 1 server after a second call, got %d: %+v", len(servers), servers)
	}
	if servers[0].Name != "Renamed" {
		t.Errorf("expected the second call to leave the existing row untouched, got name %q", servers[0].Name)
	}
}

func TestListServers_EmptyRegistry(t *testing.T) {
	setupTestDB(t)

	servers, err := ListServers()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected an empty list before EnsureDefaultServer runs, got %+v", servers)
	}
}

func TestGetServer_NotFound(t *testing.T) {
	setupTestDB(t)

	if _, err := GetServer("nope"); err == nil {
		t.Error("expected an error for an unknown server id")
	}
}

func TestGetServer_Success(t *testing.T) {
	setupTestDB(t)
	if err := EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	s, err := GetServer(DefaultServerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.Name != "Default" {
		t.Errorf("expected name %q, got %q", "Default", s.Name)
	}
}

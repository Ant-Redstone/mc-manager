package services

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lomokwa/mc-manager/db"
)

// PermissionsSeedCandidates are checked in order every boot; the first one
// found is what gets applied. ServerDir is checked first since it's the one
// location every operator can reach the same way they already reach
// server.properties (FTP/SFTP/a hosting panel's file manager) -- the repo
// root is kept as a fallback for anyone who does have that access (local
// dev, direct SSH). Deliberately not moved or relocated anywhere: this
// always re-reads whichever file the operator can already see and edit, so
// fixing a typo (wrong username, wrong role name) is just "edit the file,
// restart" -- there's no separate "already consumed" copy that a correction
// could get silently ignored in favor of.
var PermissionsSeedCandidates = []string{
	filepath.Join(ServerDir, "permissions-seed.json"),
	"./permissions-seed.json",
}

type seedEntry struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type seedFile struct {
	Users []seedEntry `json:"users"`
}

// ApplyPermissionsSeed reads the first file it finds among
// PermissionsSeedCandidates and assigns each listed username its role --
// but only for a user who doesn't already have one. It runs on every boot
// rather than once, on purpose: the file is usually dropped before anyone
// has registered yet, so the intended owner's account may not exist on the
// first boot that sees it. Once a user has a role (assigned here or from
// the UI), this never touches them again, so hand-tuned overrides are never
// clobbered by a redeploy, and neither is a role someone already fixed by
// hand after a typo'd seed entry silently missed them.
func ApplyPermissionsSeed() {
	path, data := readFirstSeedFile()
	if data == nil {
		return
	}

	var seed seedFile
	if err := json.Unmarshal(data, &seed); err != nil {
		slog.Error("permissions seed: invalid JSON", "path", path, "err", err)
		return
	}

	for _, entry := range seed.Users {
		applySeedEntry(entry)
	}
}

func readFirstSeedFile() (path string, data []byte) {
	for _, candidate := range PermissionsSeedCandidates {
		b, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		return candidate, b
	}
	return "", nil
}

// EnsureBootstrapOwner is a last-resort safety net for the case
// permissions-seed.json wasn't dropped in place before this feature's first
// deploy: with role assignment being deny-by-default, that would otherwise
// leave every existing account (including whoever has actually been running
// this server) suddenly unable to do anything the instant it goes live. If
// literally no one has a role yet after the seed file has had its turn, the
// earliest-registered account becomes Owner -- imperfect (a second admin
// still needs assigning by hand, through the UI this unlocks), but never a
// silent total lockout. A no-op the moment anyone has a role, seeded or
// hand-assigned, so this can never override real configuration.
func EnsureBootstrapOwner() {
	var count int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM user_roles`).Scan(&count); err != nil {
		slog.Error("bootstrap owner: failed to check existing roles", "err", err)
		return
	}
	if count > 0 {
		return
	}

	var firstUserID int
	if err := db.DB.QueryRow(`SELECT id FROM users ORDER BY id ASC LIMIT 1`).Scan(&firstUserID); err != nil {
		return // no one has registered yet -- nothing to bootstrap
	}
	if err := SetUserRole(firstUserID, "Owner"); err != nil {
		slog.Error("bootstrap owner: failed to assign role", "err", err)
		return
	}
	slog.Warn("bootstrap: no roles existed yet, granted Owner to the first registered account", "user_id", firstUserID)
}

func applySeedEntry(entry seedEntry) {
	var userID int
	if err := db.DB.QueryRow(`SELECT id FROM users WHERE username = ?`, entry.Username).Scan(&userID); err != nil {
		slog.Warn("permissions seed: user not registered yet, will retry next boot", "user", entry.Username)
		return
	}

	var existing int
	err := db.DB.QueryRow(`SELECT user_id FROM user_roles WHERE user_id = ?`, userID).Scan(&existing)
	if err == nil {
		return // already has a role -- never overwritten by the seed
	}

	if _, _, _, err := GetRoleByName(entry.Role); err != nil {
		slog.Error("permissions seed: unknown role", "user", entry.Username, "role", entry.Role)
		return
	}
	if err := SetUserRole(userID, entry.Role); err != nil {
		slog.Error("permissions seed: failed to assign role", "role", entry.Role, "user", entry.Username, "err", err)
		return
	}
	slog.Info("permissions seed: role assigned", "user", entry.Username, "role", entry.Role)
}

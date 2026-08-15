package services

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lomokwa/mc-manager/db"
	"github.com/lomokwa/mc-manager/types"
)

// DefaultServerID is the id of the single row EnsureDefaultServer seeds, and
// the id DefaultRuntime() (services/runtime.go) always resolves to.
const DefaultServerID = "default"

// EnsureDefaultServer seeds the servers registry with a single row pointing
// at the server directory that already exists today (ServerDir), the first
// time this ever runs against a given database. It must never move, copy,
// or rename anything on disk -- it only records where the existing install
// already lives, which is why every value inserted below is one of the
// pre-existing constants/defaults from constants.go, not a new path.
//
// Idempotent: safe to call on every boot. A COUNT check is used (rather
// than e.g. INSERT OR IGNORE keyed on id) because the actual rule is "seed
// only an empty table" -- once any server row exists, this must never
// insert a second one, including in a later phase where other rows exist
// but "default" was somehow removed from the mix.
func EnsureDefaultServer() error {
	var count int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM servers").Scan(&count); err != nil {
		return fmt.Errorf("failed to count servers: %w", err)
	}
	if count > 0 {
		return nil
	}

	_, err := db.DB.Exec(
		`INSERT INTO servers (id, name, dir, port, jar, xms, xmx, sort)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		DefaultServerID, "Default", ServerDir, 25565, "server.jar", "1G", "2G", 0,
	)
	if err != nil {
		return fmt.Errorf("failed to insert default server: %w", err)
	}
	return nil
}

// ListServers returns every row in the registry, ordered for display (the
// sort column, then id as a stable tiebreaker).
func ListServers() ([]types.Server, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, dir, port, voice_port, jar, xms, xmx, sort, created_at
		FROM servers ORDER BY sort, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	var servers []types.Server
	for rows.Next() {
		var s types.Server
		if err := rows.Scan(&s.ID, &s.Name, &s.Dir, &s.Port, &s.VoicePort, &s.Jar, &s.Xms, &s.Xmx, &s.Sort, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan server row: %w", err)
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

// GetServer returns a single registry row by id.
func GetServer(id string) (types.Server, error) {
	var s types.Server
	row := db.DB.QueryRow(`
		SELECT id, name, dir, port, voice_port, jar, xms, xmx, sort, created_at
		FROM servers WHERE id = ?`, id)
	err := row.Scan(&s.ID, &s.Name, &s.Dir, &s.Port, &s.VoicePort, &s.Jar, &s.Xms, &s.Xmx, &s.Sort, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Server{}, fmt.Errorf("server %q not found", id)
		}
		return types.Server{}, fmt.Errorf("failed to get server %q: %w", id, err)
	}
	return s, nil
}

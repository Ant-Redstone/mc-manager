package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

//go:embed migrations.sql
var migrations string

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	return migrate()
}

func migrate() error {
	if _, err := DB.Exec(migrations); err != nil {
		return err
	}
	return ensureUserProfileColumns()
}

// ensureUserProfileColumns adds the profile-editing columns to an existing
// users table. migrations.sql only ever runs CREATE TABLE IF NOT EXISTS, so
// it can't grow a table that already exists on disk (SQLite has no ALTER
// TABLE ADD COLUMN IF NOT EXISTS) -- this runs the ALTER TABLE by hand, once,
// guarded by a PRAGMA table_info check so re-running it on every boot is a
// no-op once the column exists.
func ensureUserProfileColumns() error {
	columns := []struct{ name, definition string }{
		{"display_name", "VARCHAR(50) NOT NULL DEFAULT ''"},
		{"avatar_filename", "VARCHAR(255) NOT NULL DEFAULT ''"},
	}
	for _, col := range columns {
		exists, err := columnExists("users", col.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		// col.name/definition come from the fixed slice above, never from a
		// request, so building the statement with Sprintf carries no
		// injection risk -- ALTER TABLE doesn't support placeholders here.
		if _, err := DB.Exec(fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", col.name, col.definition)); err != nil {
			return fmt.Errorf("failed to add users.%s column: %w", col.name, err)
		}
	}
	return nil
}

func columnExists(table, column string) (bool, error) {
	rows, err := DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

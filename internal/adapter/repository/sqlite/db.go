// Package sqlite is a "frameworks & drivers" + "interface adapters"
// component: it owns the SQLite connection and implements
// usecase.ItemRepository, translating between domain.Item and rows. No
// other layer imports database/sql or modernc.org/sqlite — only this
// package does.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// migrations holds the versioned schema. Add a new file
// NNNNN_description.sql for every schema change; never edit one that has
// already been deployed.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Open connects to the SQLite database at path and applies any pending
// migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite handles one writer at a time; a single connection avoids
	// "database is locked" errors under concurrent requests.
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}
	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		log.Printf("applied migration %s", r.Source.Path)
	}
	return nil
}

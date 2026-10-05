package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/gustavogmartinelli/gtd/internal/domain"
	_ "modernc.org/sqlite"
)

func TestOpenAppliesMigrationsToFreshDB(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	item, err := domain.NewItem("Buy milk", "")
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	repo := NewItemRepository(db)
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), item.ID); err != nil {
		t.Fatalf("FindByID: %v", err)
	}
}

// A database created by the pre-migration schema must open without
// errors and keep its rows.
func TestOpenAdoptsPreMigrationDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gtd.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'inbox',
			context TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL, completed_at TEXT);
		INSERT INTO items (title, created_at, updated_at)
			VALUES ('old', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');`)
	legacy.Close()
	if err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows after migration = %d, want 1", n)
	}
}

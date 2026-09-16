// Package store provides SQLite-backed persistence for GTD items.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gustavogmartinelli/gtd/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS items (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	title        TEXT NOT NULL,
	notes        TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'inbox',
	context      TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL,
	completed_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_items_status ON items(status);
`

type Store struct {
	db *sql.DB
}

// Open creates (if needed) and connects to the SQLite database at path,
// applying the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite handles one writer at a time; a single connection avoids
	// "database is locked" errors under concurrent requests.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

var ErrNotFound = fmt.Errorf("item not found")

// Capture inserts a new item into the inbox.
func (s *Store) Capture(ctx context.Context, title, notes string) (*model.Item, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO items (title, notes, status, context, created_at, updated_at)
		 VALUES (?, ?, ?, '', ?, ?)`,
		title, notes, model.StatusInbox, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert item: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get insert id: %w", err)
	}
	return s.Get(ctx, id)
}

// List returns items, optionally filtered by status. An empty status
// returns all non-completed items.
func (s *Store) List(ctx context.Context, status string, includeCompleted bool) ([]*model.Item, error) {
	query := `SELECT id, title, notes, status, context, created_at, updated_at, completed_at FROM items WHERE 1=1`
	var args []any
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if !includeCompleted {
		query += ` AND completed_at IS NULL`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []*model.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Get returns a single item by id.
func (s *Store) Get(ctx context.Context, id int64) (*model.Item, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, notes, status, context, created_at, updated_at, completed_at
		 FROM items WHERE id = ?`, id)
	item, err := scanItem(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	return item, nil
}

// ProcessToNext moves an inbox item to the next-actions list, optionally
// assigning it a context (e.g. "@home") and rewriting its notes.
func (s *Store) ProcessToNext(ctx context.Context, id int64, gtdContext, notes string) (*model.Item, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE items SET status = ?, context = ?, notes = ?, updated_at = ? WHERE id = ?`,
		model.StatusNext, gtdContext, notes, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("process item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Update applies partial field changes to an item. Nil fields are left
// unchanged.
func (s *Store) Update(ctx context.Context, id int64, title, notes, status, gtdContext *string) (*model.Item, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if title != nil {
		item.Title = *title
	}
	if notes != nil {
		item.Notes = *notes
	}
	if status != nil {
		item.Status = *status
	}
	if gtdContext != nil {
		item.Context = *gtdContext
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET title = ?, notes = ?, status = ?, context = ?, updated_at = ? WHERE id = ?`,
		item.Title, item.Notes, item.Status, item.Context, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}
	return s.Get(ctx, id)
}

// Complete marks an item as done.
func (s *Store) Complete(ctx context.Context, id int64) (*model.Item, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE items SET completed_at = ?, updated_at = ? WHERE id = ?`,
		now, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("complete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes an item permanently.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(row rowScanner) (*model.Item, error) {
	var item model.Item
	var createdAt, updatedAt string
	var completedAt sql.NullString

	if err := row.Scan(&item.ID, &item.Title, &item.Notes, &item.Status, &item.Context,
		&createdAt, &updatedAt, &completedAt); err != nil {
		return nil, err
	}

	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if completedAt.Valid {
		t, err := time.Parse(time.RFC3339, completedAt.String)
		if err == nil {
			item.CompletedAt = &t
		}
	}
	return &item, nil
}

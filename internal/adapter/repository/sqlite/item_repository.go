package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gustavogmartinelli/gtd/internal/domain"
	"github.com/gustavogmartinelli/gtd/internal/usecase"
)

// ItemRepository implements usecase.ItemRepository against SQLite. The
// use case layer only ever sees the ItemRepository interface; this
// concrete type is wired in by the composition root (cmd/gtd-server).
type ItemRepository struct {
	db *sql.DB
}

var _ usecase.ItemRepository = (*ItemRepository)(nil)

func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Save(ctx context.Context, item *domain.Item) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO items (title, notes, status, context, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.Title, item.Notes, item.Status, item.Context,
		item.CreatedAt.Format(time.RFC3339), item.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert item: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get insert id: %w", err)
	}
	item.ID = id
	return nil
}

func (r *ItemRepository) FindByID(ctx context.Context, id int64) (*domain.Item, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, notes, status, context, created_at, updated_at, completed_at
		 FROM items WHERE id = ?`, id)
	item, err := scanItem(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	return item, nil
}

func (r *ItemRepository) FindAll(ctx context.Context, filter usecase.ListFilter) ([]*domain.Item, error) {
	query := `SELECT id, title, notes, status, context, created_at, updated_at, completed_at FROM items WHERE 1=1`
	var args []any
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if !filter.IncludeCompleted {
		query += ` AND completed_at IS NULL`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []*domain.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE items SET title = ?, notes = ?, status = ?, context = ?, updated_at = ?, completed_at = ?
		 WHERE id = ?`,
		item.Title, item.Notes, item.Status, item.Context,
		item.UpdatedAt.Format(time.RFC3339), formatNullTime(item.CompletedAt), item.ID,
	)
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(row rowScanner) (*domain.Item, error) {
	var item domain.Item
	var status, createdAt, updatedAt string
	var completedAt sql.NullString

	if err := row.Scan(&item.ID, &item.Title, &item.Notes, &status, &item.Context,
		&createdAt, &updatedAt, &completedAt); err != nil {
		return nil, err
	}

	item.Status = domain.Status(status)
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if completedAt.Valid {
		if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			item.CompletedAt = &t
		}
	}
	return &item, nil
}

func formatNullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

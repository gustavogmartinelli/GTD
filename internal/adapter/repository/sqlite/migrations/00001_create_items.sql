-- +goose Up
-- IF NOT EXISTS keeps this safe on databases created before migrations
-- were tracked: goose records it as applied without touching the data.
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

-- +goose Down
DROP INDEX IF EXISTS idx_items_status;
DROP TABLE IF EXISTS items;

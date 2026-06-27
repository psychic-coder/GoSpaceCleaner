package journal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Entry is one recorded deletion.
type Entry struct {
	ID          int64
	OriginalPath string
	TrashPath    string // where it actually went (~/.Trash/gospace-<id>-<basename>)
	SizeBytes    int64
	Kind         string
	DeletedAt    time.Time
	Restored     bool
}

// Journal wraps a local SQLite DB that records every deletion gospace performs,
// so `gospace undo` can move things back out of Trash instead of them being gone forever.
type Journal struct {
	db *sql.DB
}

func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".gospace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "journal.db"), nil
}

func Open() (*Journal, error) {
	path, err := defaultDBPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening journal db: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS deletions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_path TEXT NOT NULL,
		trash_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		kind TEXT NOT NULL,
		deleted_at TIMESTAMP NOT NULL,
		restored BOOLEAN NOT NULL DEFAULT 0
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &Journal{db: db}, nil
}

func (j *Journal) Close() error {
	return j.db.Close()
}

// Record logs a completed deletion.
func (j *Journal) Record(originalPath, trashPath string, sizeBytes int64, kind string) (int64, error) {
	res, err := j.db.Exec(
		`INSERT INTO deletions (original_path, trash_path, size_bytes, kind, deleted_at, restored)
		 VALUES (?, ?, ?, ?, ?, 0)`,
		originalPath, trashPath, sizeBytes, kind, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("recording journal entry: %w", err)
	}
	return res.LastInsertId()
}

// MarkRestored flags an entry as restored after `gospace undo` moves it back.
func (j *Journal) MarkRestored(id int64) error {
	_, err := j.db.Exec(`UPDATE deletions SET restored = 1 WHERE id = ?`, id)
	return err
}

// Recent returns the last n non-restored entries, most recent first —
// what `gospace undo` shows you to pick from.
func (j *Journal) Recent(n int) ([]Entry, error) {
	rows, err := j.db.Query(
		`SELECT id, original_path, trash_path, size_bytes, kind, deleted_at, restored
		 FROM deletions
		 WHERE restored = 0
		 ORDER BY deleted_at DESC
		 LIMIT ?`, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.OriginalPath, &e.TrashPath, &e.SizeBytes, &e.Kind, &e.DeletedAt, &e.Restored); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// TotalReclaimed sums size_bytes across all non-restored entries —
// the lifetime "space you've actually gotten back" metric.
func (j *Journal) TotalReclaimed() (int64, error) {
	var total int64
	err := j.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM deletions WHERE restored = 0`).Scan(&total)
	return total, err
}

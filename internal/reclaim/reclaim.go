package reclaim

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gospace/internal/detector"
	"gospace/internal/journal"
)

// Reclaimer performs the actual cleanup. It never calls os.RemoveAll directly
// on a candidate — everything goes through ~/.Trash first, journaled, so
// `gospace undo` has something to restore. True permanent deletion only
// happens when the user empties their system Trash, same as Finder.
type Reclaimer struct {
	journal *journal.Journal
}

func New(j *journal.Journal) *Reclaimer {
	return &Reclaimer{journal: j}
}

// Reclaim moves a candidate to Trash and records it. Returns bytes freed.
func (r *Reclaimer) Reclaim(c *detector.Candidate) (int64, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}

	trashDir := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return 0, fmt.Errorf("ensuring trash dir: %w", err)
	}

	// Namespace the trashed item so collisions (two "node_modules" dirs) don't clobber each other.
	trashName := fmt.Sprintf("gospace-%d-%s", time.Now().UnixNano(), filepath.Base(c.Path))
	trashPath := filepath.Join(trashDir, trashName)

	if err := os.Rename(c.Path, trashPath); err != nil {
		return 0, fmt.Errorf("moving %s to trash: %w", c.Path, err)
	}

	if _, err := r.journal.Record(c.Path, trashPath, c.SizeBytes, c.Kind); err != nil {
		// The move already succeeded — don't fail the whole operation over a
		// journal write error, but surface it loudly since undo won't work for this one.
		return c.SizeBytes, fmt.Errorf("WARNING: moved to trash but failed to journal (undo won't work for this item): %w", err)
	}

	return c.SizeBytes, nil
}

// Undo restores a journaled entry from Trash back to its original location.
func (r *Reclaimer) Undo(e journal.Entry) error {
	if _, err := os.Stat(e.TrashPath); err != nil {
		return fmt.Errorf("trash item missing (likely emptied already): %w", err)
	}

	if err := os.Rename(e.TrashPath, e.OriginalPath); err != nil {
		return fmt.Errorf("restoring %s: %w", e.OriginalPath, err)
	}

	return r.journal.MarkRestored(e.ID)
}

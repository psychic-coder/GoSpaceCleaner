package detector

import (
	"os"
	"path/filepath"
	"time"
)

// Candidate represents a directory flagged as a potential cleanup target.
type Candidate struct {
	Path         string    // absolute path to the directory
	SizeBytes    int64     // total size on disk
	Kind         string    // detector name, e.g. "node_modules", "xcode_derived_data"
	LastAccessed time.Time // most recent atime/mtime found in the tree
	Regenerable  bool      // true if this is trivially rebuildable (npm install, pod install, etc.)
	Reason       string    // human-readable justification shown to the user
}

// Detector is the plugin contract every cleanup rule must implement.
// Scan is called once per directory the scanner visits (not recursively —
// the scanner handles tree-walking; detectors just answer "is *this* node interesting?").
type Detector interface {
	// Name returns the detector's identifier, used as Candidate.Kind.
	Name() string

	// Match decides if `path` (a directory) is something this detector cares about.
	// It should be fast — no recursive size calculation here, just a structural check
	// (file existence, name pattern, sibling files like package.json).
	Match(path string, info os.FileInfo) bool

	// Inspect is called only on matched paths. It computes size, last-accessed,
	// and decides regenerability. This is allowed to be slower since it only
	// runs on already-matched candidates.
	Inspect(path string) (*Candidate, error)
}

// dirSize walks a directory and sums file sizes. Shared helper for detectors.
func dirSize(root string) (int64, time.Time, error) {
	var total int64
	var latest time.Time

	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip permission errors etc. rather than failing the whole scan.
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})

	return total, latest, err
}

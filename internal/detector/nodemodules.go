package detector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// NodeModulesDetector flags node_modules directories that are safe to delete:
// regenerable via `npm install` / `pnpm install` / `yarn`, and not touched recently.
type NodeModulesDetector struct {
	// StaleAfter is how long since last access before we consider it "cold."
	// Cold + regenerable = high confidence candidate.
	StaleAfter time.Duration
}

func NewNodeModulesDetector() *NodeModulesDetector {
	return &NodeModulesDetector{StaleAfter: 30 * 24 * time.Hour} // 30 days
}

func (d *NodeModulesDetector) Name() string {
	return "node_modules"
}

func (d *NodeModulesDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	if filepath.Base(path) != "node_modules" {
		return false
	}
	// Confirm the parent directory actually looks like a JS/TS project,
	// not some unrelated folder that happens to be named node_modules.
	parent := filepath.Dir(path)
	_, err := os.Stat(filepath.Join(parent, "package.json"))
	return err == nil
}

func (d *NodeModulesDetector) Inspect(path string) (*Candidate, error) {
	size, lastAccessed, err := dirSize(path)
	if err != nil {
		return nil, fmt.Errorf("inspecting %s: %w", path, err)
	}

	parent := filepath.Dir(path)
	stale := time.Since(d.gitOrMtimeReference(parent)) > d.StaleAfter

	reason := "regenerable via package manager install"
	if !stale {
		reason += "; parent project touched recently — review before deleting"
	}

	return &Candidate{
		Path:         path,
		SizeBytes:    size,
		Kind:         d.Name(),
		LastAccessed: lastAccessed,
		Regenerable:  true,
		Reason:       reason,
	}, nil
}

// gitOrMtimeReference prefers the last commit time in the parent project's
// git repo (if any) as the "is this project active" signal, since editors
// can touch mtimes without real work happening. Falls back to dir mtime.
func (d *NodeModulesDetector) gitOrMtimeReference(projectDir string) time.Time {
	gitDir := filepath.Join(projectDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		cmd := exec.Command("git", "-C", projectDir, "log", "-1", "--format=%ct")
		out, err := cmd.Output()
		if err == nil {
			var unixTs int64
			if _, scanErr := fmt.Sscanf(string(out), "%d", &unixTs); scanErr == nil {
				return time.Unix(unixTs, 0)
			}
		}
	}

	if info, err := os.Stat(projectDir); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

package detector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// HomebrewCacheDetector flags Homebrew's download cache, which accumulates
// old .bottle / .tar.gz installers that are never needed again post-install.
type HomebrewCacheDetector struct{}

func NewHomebrewCacheDetector() *HomebrewCacheDetector {
	return &HomebrewCacheDetector{}
}

func (d *HomebrewCacheDetector) Name() string {
	return "homebrew_cache"
}

func (d *HomebrewCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	cacheRoot := filepath.Join(home, "Library", "Caches", "Homebrew")
	return path == cacheRoot
}

func (d *HomebrewCacheDetector) Inspect(path string) (*Candidate, error) {
	size, lastAccessed, err := dirSize(path)
	if err != nil {
		return nil, fmt.Errorf("inspecting %s: %w", path, err)
	}

	return &Candidate{
		Path:         path,
		SizeBytes:    size,
		Kind:         d.Name(),
		LastAccessed: lastAccessed,
		Regenerable:  true,
		Reason:       "Homebrew install cache — safe to clear, prefer `brew cleanup -s` over raw delete",
	}, nil
}

// DockerReclaimDetector doesn't walk the filesystem at all — it shells out to
// `docker system df` since Docker's reclaimable space lives inside its VM disk
// image, not as plain files Go can stat directly.
type DockerReclaimDetector struct{}

func NewDockerReclaimDetector() *DockerReclaimDetector {
	return &DockerReclaimDetector{}
}

func (d *DockerReclaimDetector) Name() string {
	return "docker_reclaimable"
}

// Match is special-cased: the scanner calls this once at the $HOME root only,
// since Docker reclaim isn't filesystem-path-based.
func (d *DockerReclaimDetector) Match(path string, info os.FileInfo) bool {
	home, _ := os.UserHomeDir()
	return path == home
}

func (d *DockerReclaimDetector) Inspect(path string) (*Candidate, error) {
	cmd := exec.Command("docker", "system", "df", "--format", "{{.Reclaimable}}")
	out, err := cmd.Output()
	if err != nil {
		// Docker not installed or not running — not an error, just nothing to report.
		return nil, nil
	}

	var totalBytes int64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		totalBytes += parseDockerSize(line)
	}

	if totalBytes == 0 {
		return nil, nil
	}

	return &Candidate{
		Path:        "docker (managed by Docker daemon, not a filesystem path)",
		SizeBytes:   totalBytes,
		Kind:        d.Name(),
		Regenerable: true,
		Reason:      "reclaimable via `docker system prune` — stopped containers, dangling images, unused volumes",
	}, nil
}

// parseDockerSize converts strings like "1.2GB" or "340MB" to bytes.
func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" {
		return 0
	}

	units := map[string]float64{
		"B":  1,
		"KB": 1 << 10,
		"MB": 1 << 20,
		"GB": 1 << 30,
		"TB": 1 << 40,
	}

	for suffix, mult := range units {
		if strings.HasSuffix(s, suffix) {
			numPart := strings.TrimSuffix(s, suffix)
			val, err := strconv.ParseFloat(numPart, 64)
			if err != nil {
				return 0
			}
			return int64(val * mult)
		}
	}
	return 0
}

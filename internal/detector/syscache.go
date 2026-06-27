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

func (d *HomebrewCacheDetector) Inspect(path string) ([]*Candidate, error) {
	size, lastAccessed, err := dirSize(path)
	if err != nil {
		return nil, fmt.Errorf("inspecting %s: %w", path, err)
	}

	return []*Candidate{{
		Path:         path,
		SizeBytes:    size,
		Kind:         d.Name(),
		LastAccessed: lastAccessed,
		Regenerable:  true,
		Reason:       "Homebrew install cache — safe to clear, prefer `brew cleanup -s` over raw delete",
	}}, nil
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

func (d *DockerReclaimDetector) Inspect(path string) ([]*Candidate, error) {
	cmd := exec.Command("docker", "system", "df", "--format", "{{.Type}}|{{.Reclaimable}}")
	out, err := cmd.Output()
	if err != nil {
		// Docker not installed or not running — not an error, just nothing to report.
		return nil, nil
	}

	var candidates []*Candidate

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		
		typ := strings.TrimSpace(parts[0])
		reclaimableStr := strings.TrimSpace(parts[1])
		sizeBytes := parseDockerSize(reclaimableStr)

		if sizeBytes > 0 {
			cmdStr := "docker system prune"
			reason := "reclaimable via `docker system prune`"
			
			switch typ {
			case "Images":
				cmdStr = "docker image prune -a"
				reason = "reclaimable via `docker image prune -a` — unused images"
			case "Containers":
				cmdStr = "docker container prune"
				reason = "reclaimable via `docker container prune` — stopped containers"
			case "Local Volumes":
				cmdStr = "docker volume prune"
				reason = "reclaimable via `docker volume prune` — unused volumes"
			case "Build Cache":
				cmdStr = "docker builder prune"
				reason = "reclaimable via `docker builder prune` — build cache"
			}

			candidates = append(candidates, &Candidate{
				Path:        fmt.Sprintf("docker (%s)", typ),
				SizeBytes:   sizeBytes,
				Kind:        d.Name(),
				Regenerable: true,
				Reason:      reason,
				ReclaimCmd:  cmdStr,
			})
		}
	}

	return candidates, nil
}

// parseDockerSize converts strings like "1.2GB" or "340MB" to bytes.
// It strips suffixes like " (60%)" that `docker system df` sometimes adds.
func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, " ("); idx != -1 {
		s = s[:idx]
	}
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

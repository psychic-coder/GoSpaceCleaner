package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

type PipCacheDetector struct{}

func NewPipCacheDetector() *PipCacheDetector {
	return &PipCacheDetector{}
}

func (d *PipCacheDetector) Name() string {
	return "pip_cache"
}

func (d *PipCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, "Library", "Caches", "pip")
	return path == cachePath
}

func (d *PipCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Pip download cache — safe to clear, redownloaded on next pip install",
	}}, nil
}

type GradleCacheDetector struct{}

func NewGradleCacheDetector() *GradleCacheDetector {
	return &GradleCacheDetector{}
}

func (d *GradleCacheDetector) Name() string {
	return "gradle_cache"
}

func (d *GradleCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, ".gradle", "caches")
	if path == cachePath {
		return true
	}
	daemonPath := filepath.Join(home, ".gradle", "daemon")
	if path == daemonPath {
		return true
	}
	wrapperPath := filepath.Join(home, ".gradle", "wrapper", "dists")
	if path == wrapperPath {
		return true
	}
	return false
}

func (d *GradleCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Gradle caches/daemons/wrappers — safe to wipe, redownloaded by gradle wrappers",
	}}, nil
}

type MavenCacheDetector struct{}

func NewMavenCacheDetector() *MavenCacheDetector {
	return &MavenCacheDetector{}
}

func (d *MavenCacheDetector) Name() string {
	return "maven_cache"
}

func (d *MavenCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, ".m2", "repository")
	return path == cachePath
}

func (d *MavenCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Maven local repository (.m2) — safe to wipe, dependencies will be redownloaded",
	}}, nil
}

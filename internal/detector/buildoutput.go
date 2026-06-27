package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

type GoModCacheDetector struct{}

func NewGoModCacheDetector() *GoModCacheDetector {
	return &GoModCacheDetector{}
}

func (d *GoModCacheDetector) Name() string {
	return "go_mod_cache"
}

func (d *GoModCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, "go", "pkg", "mod", "cache")
	return path == cachePath
}

func (d *GoModCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Go module cache — safe to clear, redownloaded by go mod download",
		ReclaimCmd:   "go clean -modcache",
	}}, nil
}

type CargoCacheDetector struct{}

func NewCargoCacheDetector() *CargoCacheDetector {
	return &CargoCacheDetector{}
}

func (d *CargoCacheDetector) Name() string {
	return "cargo_cache"
}

func (d *CargoCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	
	cachePath := filepath.Join(home, ".cargo", "registry", "cache")
	if path == cachePath {
		return true
	}
	srcPath := filepath.Join(home, ".cargo", "registry", "src")
	if path == srcPath {
		return true
	}
	return false
}

func (d *CargoCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Cargo registry cache — safe to clear, redownloaded by cargo build",
	}}, nil
}

type BuildOutputDetector struct{}

func NewBuildOutputDetector() *BuildOutputDetector {
	return &BuildOutputDetector{}
}

func (d *BuildOutputDetector) Name() string {
	return "build_output"
}

func (d *BuildOutputDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	
	base := filepath.Base(path)
	parent := filepath.Dir(path)
	
	switch base {
	case "dist", "build", ".next", "out":
		// Check for package.json
		if _, err := os.Stat(filepath.Join(parent, "package.json")); err == nil {
			return true
		}
	case "target":
		// Check for Cargo.toml
		if _, err := os.Stat(filepath.Join(parent, "Cargo.toml")); err == nil {
			return true
		}
		// Check for pom.xml
		if _, err := os.Stat(filepath.Join(parent, "pom.xml")); err == nil {
			return true
		}
	}
	return false
}

func (d *BuildOutputDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Build output directory — safe to wipe, regenerated on next build",
	}}, nil
}

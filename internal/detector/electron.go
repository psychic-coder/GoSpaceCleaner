package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

type ElectronCacheDetector struct{}

func NewElectronCacheDetector() *ElectronCacheDetector {
	return &ElectronCacheDetector{}
}

func (d *ElectronCacheDetector) Name() string {
	return "electron_app_cache"
}

func (d *ElectronCacheDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	
	// Check Slack
	slackCache := filepath.Join(home, "Library", "Application Support", "Slack", "Cache")
	if path == slackCache {
		return true
	}
	
	// Check VS Code Cache (not workspaceStorage, just Cache)
	codeCache := filepath.Join(home, "Library", "Application Support", "Code", "Cache")
	if path == codeCache {
		return true
	}
	codeCachedData := filepath.Join(home, "Library", "Application Support", "Code", "CachedData")
	if path == codeCachedData {
		return true
	}
	
	return false
}

func (d *ElectronCacheDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Electron app cache (Slack/VS Code) — safe to clear, app redownloads/rebuilds as needed",
	}}, nil
}

type VSCodeWorkspaceStorageDetector struct{}

func NewVSCodeWorkspaceStorageDetector() *VSCodeWorkspaceStorageDetector {
	return &VSCodeWorkspaceStorageDetector{}
}

func (d *VSCodeWorkspaceStorageDetector) Name() string {
	return "vscode_workspace_storage"
}

func (d *VSCodeWorkspaceStorageDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	wsStorage := filepath.Join(home, "Library", "Application Support", "Code", "User", "workspaceStorage")
	return path == wsStorage
}

func (d *VSCodeWorkspaceStorageDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "VS Code workspace storage — mostly stale state from old projects, safe to delete",
	}}, nil
}

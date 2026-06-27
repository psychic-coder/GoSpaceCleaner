package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type NVMDetector struct{}

func NewNVMDetector() *NVMDetector {
	return &NVMDetector{}
}

func (d *NVMDetector) Name() string {
	return "nvm_versions"
}

func (d *NVMDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	nvmRoot := filepath.Join(home, ".nvm", "versions", "node")
	return path == nvmRoot
}

func (d *NVMDetector) Inspect(path string) ([]*Candidate, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	defaultAliasPath := filepath.Join(home, ".nvm", "alias", "default")
	defaultVersionBytes, _ := os.ReadFile(defaultAliasPath)
	defaultVersion := strings.TrimSpace(string(defaultVersionBytes))

	var candidates []*Candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versionName := entry.Name()
		if versionName == defaultVersion {
			continue // skip the default version
		}

		versionPath := filepath.Join(path, versionName)
		size, lastAccessed, err := dirSize(versionPath)
		if err != nil {
			continue
		}

		candidates = append(candidates, &Candidate{
			Path:         versionPath,
			SizeBytes:    size,
			Kind:         d.Name(),
			LastAccessed: lastAccessed,
			Regenerable:  true,
			Reason:       fmt.Sprintf("Old NVM Node version (%s) — not the default alias", versionName),
			// nvm is a bash function, so we need to run it via bash -i or similar, 
			// but we can just use the bash command.
			ReclaimCmd:   fmt.Sprintf("bash -c 'source ~/.nvm/nvm.sh && nvm uninstall %s'", versionName),
		})
	}

	return candidates, nil
}

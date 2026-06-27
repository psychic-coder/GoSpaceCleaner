package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

// XcodeDerivedDataDetector flags Xcode's build cache, which regrows automatically
// on next build and is almost always safe to wipe.
type XcodeDerivedDataDetector struct{}

func NewXcodeDerivedDataDetector() *XcodeDerivedDataDetector {
	return &XcodeDerivedDataDetector{}
}

func (d *XcodeDerivedDataDetector) Name() string {
	return "xcode_derived_data"
}

func (d *XcodeDerivedDataDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	derivedDataRoot := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	// Match the root itself, and we'll inspect its full contents in one shot
	// rather than matching every subdirectory individually.
	return path == derivedDataRoot
}

func (d *XcodeDerivedDataDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "Xcode build cache — regenerated automatically on next build",
	}}, nil
}

// XcodeSimulatorDetector flags unavailable/old iOS simulator runtimes, another
// common multi-GB offender that's safe to delete (Xcode redownloads if needed).
type XcodeSimulatorDetector struct{}

func NewXcodeSimulatorDetector() *XcodeSimulatorDetector {
	return &XcodeSimulatorDetector{}
}

func (d *XcodeSimulatorDetector) Name() string {
	return "xcode_simulator_devices"
}

func (d *XcodeSimulatorDetector) Match(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	home, _ := os.UserHomeDir()
	simRoot := filepath.Join(home, "Library", "Developer", "CoreSimulator", "Devices")
	return path == simRoot
}

func (d *XcodeSimulatorDetector) Inspect(path string) ([]*Candidate, error) {
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
		Reason:       "simulator device data — review individually, `xcrun simctl delete unavailable` is safer than a blanket wipe",
	}}, nil
}

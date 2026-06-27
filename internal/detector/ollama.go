package detector

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type OllamaModelDetector struct{}

func NewOllamaModelDetector() *OllamaModelDetector {
	return &OllamaModelDetector{}
}

func (d *OllamaModelDetector) Name() string {
	return "ollama_models"
}

// Match is special-cased: the scanner calls this once at the $HOME root only.
func (d *OllamaModelDetector) Match(path string, info os.FileInfo) bool {
	home, _ := os.UserHomeDir()
	return path == home
}

func (d *OllamaModelDetector) Inspect(path string) ([]*Candidate, error) {
	cmd := exec.Command("ollama", "list")
	out, err := cmd.Output()
	if err != nil {
		// Ollama not installed or not running
		return nil, nil
	}

	var candidates []*Candidate

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) <= 1 {
		return nil, nil // just header or empty
	}

	// Skip the header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		
		modelName := fields[0]
		
		// Parse size from fields (e.g. "3.8 GB" -> fields[len-2] and fields[len-1])
		// Wait, some fields might have different lengths. Let's just parse the 2nd to last and last.
		sizeStr := fields[len(fields)-2] + fields[len(fields)-1]
		if strings.Contains(sizeStr, "GB") || strings.Contains(sizeStr, "MB") || strings.Contains(sizeStr, "KB") || strings.Contains(sizeStr, "B") {
			// it's size
		} else {
			// if it doesn't end with a unit, maybe it was just one word
			sizeStr = fields[len(fields)-1]
		}
		
		sizeBytes := parseOllamaSize(sizeStr)
		if sizeBytes > 0 {
			candidates = append(candidates, &Candidate{
				Path:        fmt.Sprintf("ollama model: %s", modelName),
				SizeBytes:   sizeBytes,
				Kind:        d.Name(),
				Regenerable: true,
				Reason:      "reclaimable via `ollama rm`",
				ReclaimCmd:  fmt.Sprintf("ollama rm %s", modelName),
			})
		}
	}

	return candidates, nil
}

func parseOllamaSize(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	
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
			if numPart == "" {
				continue
			}
			val, err := strconv.ParseFloat(numPart, 64)
			if err != nil {
				return 0
			}
			return int64(val * mult)
		}
	}
	return 0
}

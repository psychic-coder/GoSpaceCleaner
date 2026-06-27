package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"gospace/internal/detector"
)

// Result holds everything a scan produced.
type Result struct {
	Candidates []*detector.Candidate
	Errors     []error
}


type Scanner struct {
	Detectors []detector.Detector
	Workers   int
	Progress  func(path string) // optional callback for UI updates
}

func New(detectors []detector.Detector) *Scanner {
	return &Scanner{
		Detectors: detectors,
		Workers:   runtime.GOMAXPROCS(0),
	}
}

type job struct {
	path string
	info os.FileInfo
}

// Scan walks `root` and returns every candidate flagged by any detector.
func (s *Scanner) Scan(root string) (*Result, error) {
	jobs := make(chan job, 256)
	resultsCh := make(chan *detector.Candidate, 256)
	errCh := make(chan error, 64)

	var wg sync.WaitGroup
	for i := 0; i < s.Workers; i++ {
		wg.Add(1)
		go s.worker(jobs, resultsCh, errCh, &wg)
	}

	result := &Result{}
	
	// Start draining channels immediately so workers don't block
	// if they find more results than the channel buffer can hold.
	var collectorWg sync.WaitGroup
	collectorWg.Add(2)
	
	go func() {
		defer collectorWg.Done()
		for c := range resultsCh {
			if c != nil {
				result.Candidates = append(result.Candidates, c)
			}
		}
	}()
	
	go func() {
		defer collectorWg.Done()
		for e := range errCh {
			result.Errors = append(result.Errors, e)
		}
	}()

	// Walk runs on the calling goroutine and feeds the job channel.
	// We skip descending into directories that already matched a detector
	// (e.g. once we've flagged a node_modules dir, no need to walk inside it).
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errCh <- err
			return nil // keep walking despite permission errors etc.
		}
		if !info.IsDir() {
			return nil
		}

		if s.Progress != nil {
			s.Progress(path)
		}

		jobs <- job{path: path, info: info}

		if shouldSkipDescend(path) {
			return filepath.SkipDir
		}
		return nil
	})

	close(jobs)

	// Wait for workers to finish, then close the results channels
	wg.Wait()
	close(resultsCh)
	close(errCh)
	
	// Wait for the collector goroutines to finish appending to the slices
	collectorWg.Wait()

	return result, walkErr
}

func (s *Scanner) worker(jobs <-chan job, results chan<- *detector.Candidate, errs chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		for _, d := range s.Detectors {
			if d.Match(j.path, j.info) {
				cands, err := d.Inspect(j.path)
				if err != nil {
					errs <- err
					continue
				}
				for _, cand := range cands {
					if cand != nil {
						results <- cand
					}
				}
			}
		}
	}
}

// shouldSkipDescend prevents the walk from wasting time inside directories
// we already know we're going to flag wholesale (e.g. node_modules can contain
// tens of thousands of files — no reason to walk every one of them twice).
func shouldSkipDescend(path string) bool {
	base := filepath.Base(path)
	skipNames := map[string]bool{
		"node_modules": true,
		".git":         true,
		"DerivedData":  true,
	}
	return skipNames[base]
}

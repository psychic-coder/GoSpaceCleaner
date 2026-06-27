# gospace

A small Go CLI for finding and safely reclaiming disk space on macOS developer machines. It scans for common dev-machine space hogs, shows you what's eating space before touching anything, and either moves things to the Trash (not `rm`) or executes context-specific cleanup commands. All trash-based deletions are fully journaled and can be restored with a single command.

## Why

Dev machines fill up quickly with regenerable junk — stale `node_modules`, Xcode's `DerivedData`, old simulator runtimes, Docker layers, package manager caches, and local LLM models. `gospace` helps you audit this bloat, estimate size breakdowns, and clear it out safely.

---

## Features

- **Concurrent Scanning**: Uses a fast, concurrent filesystem worker pool to scan your directories.
- **Progress Indicator**: Real-time display of folders scanned so you know it's working.
- **Scan Caching**: Results are cached locally at `~/.gospace/last_scan.json`. Subsequent `reclaim` commands use the cache instantly instead of re-scanning the entire disk (bypass with `--fresh`).
- **Interactive Mode**: Reclaim files interactively with `--interactive` / `-i` prompting you for each item.
- **Scriptable JSON Output**: Output raw scan results to JSON with the `--json` flag.
- **Safety First**: Non-managed files are moved to `~/.Trash` instead of being deleted permanently. A SQLite journal at `~/.gospace/journal.db` keeps track of what went where.
- **Batch Undo**: Undo the entire last reclaim operation with `gospace undo`.

---

## Install

Requires Go 1.22+ and Xcode command line tools (for SQLite CGO compilation).

```bash
# Verify requirements
brew install go
xcode-select --install   # if not already installed

# Clone & Build
git clone <this-repo>
cd gospace
go mod tidy
go build -o gospace ./cmd/gospace
```

---

## Usage

```bash
# 1. Scan for reclaimable space (dry run, does not delete)
./gospace scan

# Scan a specific directory instead of $HOME
./gospace scan --path ~/Projects

# Output results as raw JSON for scripting
./gospace scan --json

# 2. Reclaim space (interactive mode - recommended)
./gospace reclaim --confirm --interactive

# Reclaim space using the cache (skips re-scanning, deletes anything >50MB by default)
./gospace reclaim --confirm

# Force a fresh scan instead of loading the cache, and filter by size
./gospace reclaim --confirm --fresh --min-size-mb 100

# 3. Undo the last reclaim operation (restores files from Trash)
./gospace undo

# 4. View lifetime space saved via gospace
./gospace status
```

---

## What it Checks

| Kind | Target | Cleanup Action / Command |
|---|---|---|
| `node_modules` | Sibling `package.json` folders | Moved to Trash |
| `xcode_derived_data` | `~/Library/Developer/Xcode/DerivedData` | Moved to Trash |
| `xcode_simulator_devices` | `~/Library/Developer/CoreSimulator/Devices` | Moved to Trash |
| `homebrew_cache` | `~/Library/Caches/Homebrew` | Moved to Trash |
| `docker_reclaimable` | Reclaimable Docker build cache, images, containers, and volumes | Executes `docker system prune`, `docker image prune -a`, etc. |
| `nvm_versions` | Inactive Node versions in `~/.nvm/versions/node` (excl. default alias) | Executes `nvm uninstall <version>` |
| `ollama_models` | Heavyweight models listed in `ollama list` | Executes `ollama rm <model>` |
| `electron_app_cache` | Caches for Slack, VS Code Cache, VS Code CachedData | Moved to Trash |
| `vscode_workspace_storage` | `~/Library/Application Support/Code/User/workspaceStorage` | Moved to Trash |
| `pip_cache` | `~/Library/Caches/pip` | Moved to Trash |
| `gradle_cache` | Gradle caches, daemons, and wrapper downloads | Moved to Trash |
| `maven_cache` | Local Maven repository cache (`~/.m2/repository`) | Moved to Trash |
| `go_mod_cache` | Go module cache (`~/go/pkg/mod/cache`) | Executes `go clean -modcache` |
| `cargo_cache` | Cargo registry cache (`~/.cargo/registry/cache`, `~/.../src`) | Moved to Trash |
| `build_output` | Common build output folders (`dist`, `build`, `.next`, `out`, `target`) | Moved to Trash |

---

## Safety & Undo Limitations

Nothing moved to the Trash is permanently deleted until you empty the macOS Trash folder. However, please note:
> [!WARNING]
> Cleanup candidates flagged with a **custom reclaim command** (such as `docker system prune`, `nvm uninstall`, `ollama rm`, or `go clean -modcache`) are executed directly via their respective CLIs and **cannot be restored** using `gospace undo`. These items are labeled accordingly during scans.

---

## Project Layout

```
cmd/gospace/main.go       CLI Command configurations & runner
internal/scanner/         Concurrent filesystem directory walker & worker pool
internal/detector/        Detector implementations (Node, Docker, caches, etc.)
internal/journal/         SQLite-backed deletion logging
internal/reclaim/         Trash movement, CLI execution, and restoration logic
```

---

## Extending

Add a new detector by implementing the `detector.Detector` interface in `internal/detector/`:

```go
type Detector interface {
	Name() string
	Match(path string, info os.FileInfo) bool
	Inspect(path string) ([]*Candidate, error) // Returns one or more reclamation candidates
}
```

Once written, register the detector in `allDetectors()` inside [cmd/gospace/main.go](file:///Users/rohitganguly/Downloads/gospace/cmd/gospace/main.go).
# gospace

A small Go CLI for finding and safely reclaiming disk space on a Mac. Scans for common dev-machine space hogs (`node_modules`, Xcode build cache, simulator data, Homebrew cache, Docker reclaimable space), shows you what's eating space before touching anything, and moves things to Trash (not `rm`) so deletions are undoable.

## Why

Dev machines fill up fast with regenerable junk — stale `node_modules`, Xcode's `DerivedData`, old simulator runtimes, Docker layers. This tool finds it, shows you the size breakdown, and clears it without permanently deleting anything you might want back.

## Install

Requires Go 1.22+ and Xcode command line tools (for `go-sqlite3`'s cgo dependency).

```bash
brew install go
xcode-select --install   # if not already installed
```

```bash
git clone <this-repo>
cd gospace
go mod tidy
go build -o gospace ./cmd/gospace
```

## Usage

```bash
# See what's reclaimable — dry run, deletes nothing
./gospace scan

# Scan a specific directory instead of $HOME
./gospace scan --path ~/Projects

# Move flagged items to Trash (journaled, undoable)
./gospace reclaim --confirm

# Only touch things bigger than 200MB
./gospace reclaim --confirm --min-size-mb 200

# Changed your mind? Restore the last reclaimed batch
./gospace undo

# Lifetime space reclaimed via this tool
./gospace status
```

## What it checks

| Detector | What it flags |
|---|---|
| `node_modules` | Directories with a sibling `package.json`, weighted by git activity / staleness |
| `xcode_derived_data` | `~/Library/Developer/Xcode/DerivedData` — Xcode's build cache |
| `xcode_simulator_devices` | `~/Library/Developer/CoreSimulator/Devices` — old/unused simulator data |
| `homebrew_cache` | `~/Library/Caches/Homebrew` — old install downloads |
| `docker_reclaimable` | Output of `docker system df`, if Docker is installed |

## Safety

Nothing is permanently deleted by this tool. `reclaim` moves matched items into `~/.Trash` and logs each move to a local SQLite journal at `~/.gospace/journal.db`. `gospace undo` restores from there. Things are only gone for good once you empty your Mac's Trash, same as deleting in Finder.

## Project layout

```
cmd/gospace/main.go       CLI commands (scan, reclaim, undo, status)
internal/scanner/         concurrent filesystem walker
internal/detector/        detector plugins (one file per detector type)
internal/journal/         SQLite-backed deletion log
internal/reclaim/         Trash-move + restore logic
```

## Extending

Add a new detector by implementing the `detector.Detector` interface (`Name`, `Match`, `Inspect`) in `internal/detector/`, then register it in `allDetectors()` in `cmd/gospace/main.go`.
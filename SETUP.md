# gospace — setup & run on your Mac

I couldn't compile/run this in the sandbox (no Go toolchain there, and more
importantly it can't see your real `~/Library`, `node_modules`, etc. anyway —
this needs to run on your actual machine).

## 1. Extract

```bash
tar -xzf gospace.tar.gz
cd gospace
```

## 2. Install Go (if you don't have it)

```bash
brew install go
go version   # confirm 1.22+
```

## 3. Fetch dependencies

```bash
go mod tidy
```

This pulls in `cobra` (CLI framework) and `go-sqlite3` (journal DB).
Note: `go-sqlite3` uses cgo, so make sure Xcode command line tools are
installed (`xcode-select --install`) — you almost certainly already have this.

## 4. Build

```bash
go build -o gospace ./cmd/gospace
```

This gives you a single binary: `./gospace`

## 5. Use it

```bash
# Dry run — shows what's reclaimable, deletes nothing
./gospace scan

# Scan a specific path instead of $HOME
./gospace scan --path ~/Projects

# Actually move things to Trash (journaled, undoable)
./gospace reclaim --confirm

# Only touch candidates bigger than 200MB
./gospace reclaim --confirm --min-size-mb 200

# Changed your mind? Restore the last batch from Trash
./gospace undo

# See lifetime space you've reclaimed
./gospace status
```

## What it actually checks right now

- `node_modules` directories (cross-referenced against sibling `package.json`,
  with git-log-aware staleness so it doesn't flag active projects)
- Xcode `DerivedData` (build cache, fully regenerable)
- Xcode simulator device data
- Homebrew download cache
- Docker reclaimable space (via `docker system df`, if Docker is installed)

Items are moved to `~/.Trash` with a journal entry in `~/.gospace/journal.db` —
nothing is ever `rm`'d directly, so `gospace undo` works until you empty Trash.

## Extending it (this is the part worth highlighting on your resume)

Add a new detector by implementing the `detector.Detector` interface
(`Name`, `Match`, `Inspect`) in `internal/detector/`, then register it in
`allDetectors()` in `cmd/gospace/main.go`. Good next additions: pip/conda
caches, old iOS/Android emulator images, Slack/Electron app caches, `.next`/
`dist`/`target` build output directories.

## A real benchmark to run and quote

```bash
# naive sequential walk for comparison
time find ~ -name node_modules -type d 2>/dev/null

# your concurrent scanner
time ./gospace scan
```

Run both, note the wall-clock difference — that's your actual
"X% faster via concurrent directory traversal" resume metric, backed by a
number you generated yourself instead of a guessed one.

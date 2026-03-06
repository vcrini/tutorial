# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Run
go run .

# Run all tests
go test ./...

# Run a single test
go test -run TestSaveLoadPNGList ./...

# Lint (if golangci-lint is installed)
golangci-lint run
```

## Architecture

**lazydaggerheart** is a Go terminal UI companion app for the Daggerheart TTRPG, written in Italian. It manages player characters (PNG), encounters, monsters, equipment, and dice rolling.

### UI

The entry point (`main.go`) calls `runTViewUI()` in `tview_ui.go`. All UI logic lives there. New UI features go in `tview_ui.go`.

### File responsibilities

| File | Purpose |
|------|---------|
| `tview_ui.go` | Full tview UI — all panels (dice, PNG, encounter, monsters, environments, equipment, cards, classes, notes, treasure) |
| `data.go` | All data structs (`PNG`, `Monster`, `Environment`, `EquipmentItem`, `CardItem`, `ClassItem`) plus YAML load/save functions and embedded config FS |
| `encounter.go` | Encounter data structs and persistence helpers (`loadEncounter`, `saveEncounter`, `nextEncounterSeq`) |

### Data persistence

State is saved to `~/.lazydaggerheart/` at runtime (path set by `initStoragePaths()` in `data.go`):
- `pngs.yml` — player character list and selected PNG
- `encounter.yml` — active encounter monsters and wound state
- `state.yml` — fear counter
- `notes.yml` — session notes

### Embedded config files

The `config/` directory is compiled into the binary via `//go:embed` in `data.go`. `readData()` checks the `config/` prefix and reads from the embedded FS first, falling back to disk:
- `config/mostri.yml` — monster database
- `config/ambienti.yml` — environments
- `config/equipaggiamento.yaml` — equipment
- `config/carte.yaml` — spell/ability cards
- `config/classi.yaml` — character classes
- `config/names.yaml` — name lists for random PNG generation

### Focus system in tview_ui.go

Panel focus is tracked as an `int` constant (e.g. `focusDice`, `focusPNG`, `focusMonList`, etc.). The tview `InputCapture` on the `pages` primitive routes keyboard events to the correct panel handler based on the current focus value.


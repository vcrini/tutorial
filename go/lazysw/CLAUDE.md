# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Run (TUI mode)
go run .

# Run (CLI/headless mode)
./lazysw cli dice "2d6+1"
./lazysw cli monsters [filter] [--source core,iz]

# Test
go test ./...
```

## Architecture

This is a terminal application for managing **Savage Worlds Adventure Edition (SWADE)** tabletop RPG sessions. It has two UI modes:

- **TUI mode** (default): built with `tview` (rivo/tview). Entry point: `runTViewUI()` in `tview_ui.go`.
- **CLI/headless mode**: activated via `./lazysw cli ...`. Entry point: `runCLI()` in `cli.go`.

### Key files

| File | Purpose |
|---|---|
| `main.go` | Entrypoint; routes to TUI or CLI mode |
| `tview_ui.go` | Active TUI (tview): panels, keybindings, layout, all UI logic |
| `cli.go` | Headless CLI mode (dice rolls, monster listing) |
| `data.go` | All data types (`Monster`, `PNG`, `EquipmentItem`, `ClassItem`, etc.) and load/save functions |
| `encounter.go` | Encounter domain logic (wounds, conditions, initiative, persistence) |
| `main_test.go` | Unit tests |

### Data & persistence

- **Config files** (`config/*.yml`/`.yaml`): read-only game data (monsters, equipment, classes, names). Versioned in the repo.
- **State files**: persisted to `~/.lazysw/` (or `$LAZYSW_HOME/`). Managed via `persistentPath()` in `data.go`:
  - `pngs.yml` — NPC list
  - `encounter.yml` — current encounter state
  - `dice_history.yml` — dice roll history (capped at 200 entries)
- Legacy JSON format for `pngs.yml` is supported via `readPersistentFileWithFallback()`.

### Dice rolling

Uses the external module `github.com/vcrini/diceroll`. Supports Savage Worlds `D*` notation (exploding dice, wild die, advantage/disadvantage).

### TUI panels (tview)

The active UI (`view.go`) organizes content into panels navigable with `tab`/`1`/`2`/`3`:
- Panel 0: PNGs (NPCs)
- Panel 1: Encounter
- Panel 2: Monsters catalog

Filters on panels use `/` for raw search, `u`/`t`/`g` for structured filters.

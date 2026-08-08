# Splorer: Go TUI File Explorer (MVB)

## Context
A minimal file explorer built in Go as a terminal UI (TUI) with mouse support. The MVB supports browsing directories and opening files. Additional features (search, etc.) will be added later.

## Library Choice
**Bubble Tea v2** (`charm.land/bubbletea/v2`) + **Lipgloss v2** (`charm.land/lipgloss/v2`)

Reasons: first-class mouse support via `MouseClickMsg`/`MouseWheelMsg`, clean Elm-style update loop, active ecosystem. No third-party file library needed — use stdlib `os` and `path/filepath`.

## Project Structure
```
/home/ben/splorer/
├── go.mod
├── main.go
└── internal/
    ├── app/
    │   └── app.go          # root Model: composes panes, handles WindowSizeMsg
    ├── filetree/
    │   ├── model.go        # FileTree model: dir listing, cursor, mouse
    │   ├── model_test.go
    │   ├── item.go         # FileEntry struct
    │   └── item_test.go
    └── opener/
        ├── opener.go       # OpenFile() via xdg-open
        └── opener_test.go
```

## UI Layout
Single pane: full-width file list with a status bar at the bottom.

```
 /home/ben/projects
 ──────────────────────────────────────────────
  ▶  docs/                  dir    2026-04-01
     go.mod                 1.2 KB 2026-04-10
     main.go                4.5 KB 2026-04-12
     README.md              800 B  2026-04-01
 ──────────────────────────────────────────────
 4 items   q quit  ↑↓ navigate  enter open  ← go up
```

Directories shown first (blue, bold), files after. Selected row highlighted.

## Keyboard Bindings
| Key          | Action                        |
|--------------|-------------------------------|
| `↑` / `k`   | cursor up                     |
| `↓` / `j`   | cursor down                   |
| `Enter`      | navigate into dir or open file |
| `Backspace` / `←` / `h` | go up one directory |
| `~`          | jump to home                  |
| `PgUp`/`PgDn` | scroll by page               |
| `q` / `Ctrl+C` | quit                        |

## Mouse Support
- **Single click**: move cursor to clicked row
- **Double click**: open/navigate (same row within 500ms)
- **Scroll wheel**: move cursor up/down

## Opening Files
Uses `xdg-open` via `cmd.Start()` (non-blocking — TUI stays alive).

## Implementation Steps
1. Scaffold: `go mod init`, directory structure
2. `opener` package: `OpenFile(path string) error`
3. `FileEntry` + `loadDir`: struct, `os.ReadDir`, sort dirs first
4. `filetree` model: keyboard navigation, styled view
5. Root `app` model: compose filetree, status bar
6. `main.go`: wire up `tea.NewProgram`
7. Mouse support: click/double-click/wheel in filetree
8. Edge cases: permission errors, filename truncation, root guard

## Test Coverage
- `TestLoadDir_SortOrder`, `TestLoadDir_Empty`, `TestLoadDir_PermissionDenied`
- `TestCursorBounds_Up`, `TestCursorBounds_Down`
- `TestNavigateInto`, `TestNavigateUp`, `TestNavigateUp_AtRoot`
- `TestDoubleClickDetection`
- `TestFileEntry_Title_Dir`, `TestFileEntry_Title_File`, `TestHumanizeSize`
- `TestOpenFile_BadPath` (integration-gated with `//go:build integration`)

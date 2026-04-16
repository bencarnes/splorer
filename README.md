# splorer

A keyboard- and mouse-driven terminal file explorer built in Go.

## What it does

splorer opens in your current working directory and displays a scrollable list
of files and directories. Directories are shown first, sorted
case-insensitively, followed by files in the same order. Each row shows the
entry name, size (or "dir"), and last-modified date.

A menu bar at the top of the screen provides access to application features.
Currently it contains a single **Openers** menu that lets you configure which
program is used to open files of a given extension, overriding the system
default (`xdg-open`).

### Controls

**File tree**

| Input | Action |
|---|---|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `PgUp` / `PgDn` | Scroll one page |
| `Enter` / `→` / `l` | Open file or enter directory |
| `Backspace` / `←` / `h` | Go up to parent directory |
| `~` | Jump to home directory |
| Single click | Move cursor to row |
| Double-click | Open file or enter directory |
| Scroll wheel | Move cursor up / down |
| `q` / `Ctrl+C` | Quit |

**Menu bar**

| Input | Action |
|---|---|
| `Alt+O` | Open the Openers dialog |
| Click on a menu label | Open that menu's dialog |

**Openers dialog**

| Input | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate the association list |
| `d` | Delete the selected association |
| `Tab` | Cycle focus: list → Ext field → Program field → list |
| `Shift+Tab` | Cycle focus in reverse |
| `Enter` (in Program field) | Add the new association |
| `Esc` | Close the dialog (changes are saved automatically) |

### File associations

Associations are stored in `~/.config/splorer/openers.json` and are loaded at
startup. The file is a plain JSON object mapping lowercase extensions
(including the leading dot) to program names:

```json
{
  ".pdf": "evince",
  ".mp3": "vlc",
  ".png": "eog"
}
```

When a file is opened, splorer looks up its extension. If a match is found the
configured program is launched; otherwise `xdg-open` is used as the fallback.

## Building

**Prerequisites:** Go 1.25 or later.

```sh
# Enter the repo
cd splorer

# Download dependencies
go mod download

# Build the binary
go build -o splorer .

# Run
./splorer
```

To start in a specific directory, `cd` there first or install the binary on
your `$PATH`:

```sh
go install github.com/bjcarnes/splorer@latest
```

## Running tests

```sh
go test ./...
```

The `internal/opener` package also has an integration test that requires a
desktop environment with `xdg-open`. It is excluded from the default test run
and must be opted into explicitly:

```sh
go test -tags integration ./internal/opener/
```

## Project structure

```
splorer/
├── main.go
└── internal/
    ├── app/
    │   └── app.go                Root tea.Model. Owns all sub-components, routes
    │                             messages, handles menu bar activation, dispatches
    │                             file-open events using the associations map.
    ├── associations/
    │   ├── store.go              Load() / Save() — reads and writes
    │   │                         ~/.config/splorer/openers.json.
    │   ├── store_test.go
    │   ├── dialog.go             Openers dialog component: association list,
    │   │                         add-association form, text inputs, focus state.
    │   └── dialog_test.go
    ├── filetree/
    │   ├── item.go               FileEntry struct and humanizeSize helper.
    │   ├── item_test.go
    │   ├── model.go              File-tree component: directory loading, cursor/
    │   │                         scroll, keyboard and mouse handling, rendering.
    │   │                         Emits OpenFileMsg instead of calling opener directly.
    │   └── model_test.go
    ├── menubar/
    │   ├── menubar.go            MenuBar and Item types. Items are activated by
    │   │                         keyboard hotkey or mouse click. Designed to accept
    │   │                         additional items and future dropdown sub-menus.
    │   └── menubar_test.go
    └── opener/
        ├── opener.go             OpenFile() (xdg-open) and OpenFileWith(path, prog).
        ├── opener_test.go
        └── opener_integration_test.go   (build tag: integration)
```

## Key design decisions

- **`filetree.Model` is not a `tea.Model`.**  
  Its `Update` method returns `(filetree.Model, tea.Cmd)` — a concrete type, not
  an interface. This makes tests straightforward (no casts), keeps the component
  easy to reason about, and prevents accidental use as a root model. The same
  convention applies to `associations.Dialog` and `menubar.MenuBar`. Only
  `app.Model` implements `tea.Model`.

- **`filetree` emits `OpenFileMsg` instead of opening files itself.**  
  When the user activates a file the filetree returns a `func() tea.Msg` command
  that yields `filetree.OpenFileMsg{Path: ...}`. `app.Update` receives this and
  decides which program to use (checking the associations map first, falling back
  to `xdg-open`). This keeps file-opener logic out of the file browser.

- **Menu bar items carry a `tea.Msg` value, not a callback.**  
  Each `menubar.Item` stores a `Msg tea.Msg`. `HandleKey` / `HandleClick` wrap it
  in a `tea.Cmd` and return it; the event loop dispatches it naturally. Adding a
  new menu item means adding one `Item` to the slice in `app.New` — no other
  wiring required.

- **Mouse Y-coordinate translation happens in `app.Update`.**  
  The menu bar occupies `menuBarHeight = 1` terminal row. When a `MouseClickMsg`
  arrives with `Y >= menuBarHeight`, `app.Update` subtracts `menuBarHeight`
  before forwarding it to the filetree. The filetree therefore always sees Y=0 as
  its own top row, and its `headerHeight` constant remains accurate.

- **The Openers dialog is a full-screen overlay.**  
  When `dialogOpen == true`, `app.Update` routes all events to `dialog.Update`
  exclusively, and `app.View()` renders the dialog content in place of the
  filetree. The menu bar is always rendered above the active content.

- **Associations are saved on every dialog close.**  
  `app.Update` calls `associations.Save` each time the dialog is dismissed.
  Failures are ignored (best-effort). Changes made in the dialog are live
  immediately — there is no explicit "cancel" that discards edits.

- **Double-click is implemented manually.**  
  Bubble Tea v2 has no built-in double-click. The filetree records the time and
  entry index of the last click; a second click on the same row within 500 ms
  triggers open/navigate.

## Adding features

**New menu items**: add an `Item` to the slice in `app.New`, define a message
type in `app.go`, and handle it in `app.Update`. No changes to `menubar` are
needed.

**New key bindings** in the file tree: add a case to the `tea.KeyPressMsg`
switch in `filetree/model.go:Update`. If the binding should be interceptable by
the menu bar (e.g. a global shortcut), add it to `app.Update` instead, before
the `m.filetree.Update(msg)` call.

**Layout changes** (e.g. a preview pane): update `filetree.Render()` and adjust
`headerHeight` / `footerHeight` if the number of fixed rows changes, since those
constants drive mouse hit-testing.

**Tests** for logic in any component can be written against the concrete model
type directly. Construct the model, call `Update` with synthetic `tea.KeyPressMsg`
or `tea.MouseClickMsg` values, and inspect the returned model. See
`filetree/model_test.go` and `associations/dialog_test.go` for patterns.

## Dependencies

| Package | Role |
|---|---|
| `charm.land/bubbletea/v2` | TUI event loop (Elm-style model/update/view) |
| `charm.land/lipgloss/v2` | Terminal styling (bold, colour, reverse highlight) |

All other entries in `go.mod` are transitive dependencies pulled in by the two
above.

# splorer

A keyboard- and mouse-driven terminal file explorer built in Go.

## What it does

splorer opens in your current working directory and displays a scrollable list
of files and directories. Directories are shown first, sorted
case-insensitively, followed by files in the same order. Each row shows the
entry name, size (or "dir"), and last-modified date.

A menu bar at the top of the screen provides access to application features.
It contains a **Find** menu for searching files by name, an **Openers** menu
for configuring which program opens files of a given extension (overriding the
system default — `xdg-open` on Linux, `start` on Windows), a **Bookmarks** menu
for saving and navigating to frequently-used files and directories, and a
**Sort** menu for changing the order in which directory entries are listed.

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
| `q` / `Esc` | Quit |

**Menu bar**

| Input | Action |
|---|---|
| `Ctrl+F` | Open the Find / search view |
| `Ctrl+B` | Open the Create Bookmark dialog for the selected entry |
| `Alt+F` | Open the Find / search view via the menu |
| `Alt+O` | Open the Openers dialog |
| `Alt+B` | Open the Bookmarks page |
| `Alt+S` | Open the Sort dialog |
| Click on a menu label | Open that menu's dialog |

**Find / search view**

| Input | Action |
|---|---|
| Typing | Update the search pattern |
| `Enter` | Start the search |
| `←` / `→` | Move cursor within the pattern field |
| `↑` / `k` | Move cursor up in results |
| `↓` / `j` | Move cursor down in results |
| `PgUp` / `PgDn` | Scroll results one page |
| `Enter` / `→` / `l` | Open file or navigate into directory |
| Single click | Move cursor to result row |
| Double-click | Open file or navigate into directory |
| Scroll wheel | Move cursor up / down in results |
| `Esc` | Cancel a running search, or close the search view |
| `Backspace` | Delete a character (input phase); close the search view (results phase) |

While a search is running a progress counter is shown. The search is recursive:
it looks inside the directory currently shown in the file tree and all
sub-directories. Patterns follow `filepath.Match` syntax — both exact names
(`main.go`) and wildcards (`*.go`, `foo*`) are supported. Results are sorted by
their full path. Relative paths (relative to the directory being searched) are
displayed in the results list.

**Openers dialog**

| Input | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate the association list |
| `d` | Delete the selected association |
| `Tab` | Cycle focus: list → Ext field → Program field → list |
| `Shift+Tab` | Cycle focus in reverse |
| `Enter` (in Program field) | Add the new association |
| `Esc` | Close the dialog (changes are saved automatically) |

**Create Bookmark dialog** (opened with `Ctrl+B`)

| Input | Action |
|---|---|
| Typing | Enter a name for the bookmark |
| `Enter` | Save the bookmark (requires at least one character) |
| `Esc` | Cancel without saving |

**Bookmarks page** (opened with `Alt+B`)

| Input | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate the bookmark list |
| `Enter` / `→` / `l` | Open bookmark (navigate to directory or open file) |
| `Del` | Open delete confirmation dialog for the selected bookmark |
| Single click | Move cursor to row |
| Double-click | Open bookmark |
| Scroll wheel | Move cursor up / down |
| `Esc` / `Backspace` | Close the bookmarks page |

**Sort dialog** (opened with `Alt+S`)

| Input | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Move through the four sort orders |
| `Enter` | Apply the selected sort order and close |
| `Esc` | Cancel without changing the sort order |

The active sort order is shown in the top-right corner of the file tree.
Sort orders: **Name** (alphabetical, default), **Timestamp** (newest first),
**Size** (largest first), **Type** (by file extension, then name).
Directories always appear before files regardless of sort order.

**Delete confirmation dialog** (inside the Bookmarks page)

| Input | Action |
|---|---|
| `y` / `Y` | Confirm deletion |
| `n` / `N` | Cancel |
| `Esc` | Cancel |

### Bookmarks

Bookmarks are stored in `~/.config/splorer/bookmarks.json` and are loaded at
startup. Each bookmark has a name and a path (file or directory). Press
`Ctrl+B` from the file tree to bookmark the currently selected entry; the
dialog shows the path and lets you type a name. Changes are saved
automatically when the dialog closes.

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
configured program is launched; otherwise the platform default is used as the
fallback — `xdg-open` on Linux, the `start` shell builtin (via `cmd /c`) on
Windows.

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
desktop environment with the platform opener (`xdg-open` on Linux, `start` on
Windows). It is excluded from the default test run and must be opted into
explicitly:

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
    │                             file-open and directory-navigate events.
    ├── associations/
    │   ├── store.go              Load() / Save() — reads and writes
    │   │                         ~/.config/splorer/openers.json.
    │   ├── store_test.go
    │   ├── dialog.go             Openers dialog component: association list,
    │   │                         add-association form, text inputs, focus state.
    │   └── dialog_test.go
    ├── bookmarks/
    │   ├── store.go              Load() / Save() — reads and writes
    │   │                         ~/.config/splorer/bookmarks.json.
    │   ├── store_test.go
    │   ├── create.go             CreateDialog component: name input, path display,
    │   │                         OK/Cancel, saves on Enter (≥1 char), Esc cancels.
    │   ├── create_test.go
    │   ├── page.go               Bookmarks list page: navigation, activation
    │   │                         (NavigateDirMsg for dirs, OpenFileMsg for files),
    │   │                         inline delete-confirmation dialog, mouse support.
    │   └── page_test.go
    ├── sortdialog/
    │   ├── dialog.go             Sort-order picker: four options (Name / Timestamp /
    │   │                         Size / Type), arrow-key navigation, Enter confirms,
    │   │                         Esc cancels. Chosen() returns filetree.SortOrder.
    │   └── dialog_test.go
    ├── filetree/
    │   ├── item.go               FileEntry struct and humanizeSize helper.
    │   ├── item_test.go
    │   ├── model.go              File-tree component: directory loading, cursor/
    │   │                         scroll, keyboard and mouse handling, rendering.
    │   │                         Emits OpenFileMsg instead of calling opener directly.
    │   ├── model_test.go
    │   └── sort.go               SortOrder type, Label(), AllSortOrders, sortGroup().
    │                             Dirs always precede files; order within each group
    │                             is controlled by the active SortOrder.
    ├── menubar/
    │   ├── menubar.go            MenuBar and Item types. Items are activated by
    │   │                         keyboard hotkey or mouse click. Designed to accept
    │   │                         additional items and future dropdown sub-menus.
    │   └── menubar_test.go
    ├── opener/
    │   ├── opener.go             OpenFileWith(path, prog) — shared across platforms.
    │   ├── opener_unix.go        OpenFile() via xdg-open          (build tag: !windows)
    │   ├── opener_windows.go     OpenFile() via cmd /c start "" … (build tag: windows)
    │   ├── opener_test.go
    │   └── opener_integration_test.go   (build tag: integration)
    └── search/
        ├── model.go              Find / search view: text-input phase, background
        │                         recursive walk (filepath.WalkDir + context
        │                         cancellation), streaming result batches via channel,
        │                         sorted results list, keyboard/mouse navigation.
        │                         Emits OpenFileMsg (files) and NavigateDirMsg (dirs).
        └── model_test.go
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
  to the platform default: `xdg-open` on Linux, `start` on Windows). This keeps
  file-opener logic out of the file browser.

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

- **The Find view and Openers dialog are full-screen overlays.**  
  When `searchOpen` or `dialogOpen` is `true`, `app.Update` routes all events
  to that component exclusively, and `app.View()` renders it in place of the
  filetree. The menu bar is always rendered above the active content.

- **Search runs in a background goroutine with context cancellation.**  
  `search.startSearch` launches a `filepath.WalkDir` goroutine and returns a
  `waitForBatch` command that blocks on a channel. Each batch of up to 100
  results is sent to the Tea event loop, which appends them to the model and
  issues the next `waitForBatch`. Pressing `Esc` or `Backspace` calls the
  context cancel function; `defer close(ch)` in the goroutine unblocks any
  pending `waitForBatch`. Stale messages are discarded via a per-search session
  ID so opening a new search while the previous goroutine is still winding down
  is safe.

- **File vs. directory activation from search results.**  
  Selecting a file emits `filetree.OpenFileMsg` (keeping the search view open
  so the user can continue browsing). Selecting a directory emits
  `search.NavigateDirMsg`; `app.Update` navigates the underlying file tree to
  that path and closes the search view.

- **Associations are saved on every dialog close.**  
  `app.Update` calls `associations.Save` each time the dialog is dismissed.
  Failures are ignored (best-effort). Changes made in the dialog are live
  immediately — there is no explicit "cancel" that discards edits.

- **Sort order is stored in the filetree model and applied on every directory load.**  
  `filetree.Model` holds a `sortOrder SortOrder` field (default `SortByName`).
  `loadDir` passes it to `sortGroup`, which sorts directories and files
  independently so directories always appear first. `SetSortOrder` reloads the
  current directory with the new order and resets the cursor to 0. The active
  sort order label is displayed in the top-right corner of the file tree header.

- **Bookmarks follow the same save-on-close pattern.**  
  `app.Update` calls `bookmarks.Save` when the create dialog closes with a
  saved name, and again when the bookmarks page closes (in case entries were
  deleted). The `Page` operates on a copy of the bookmark slice; the caller
  reads it back via `Page.Bookmarks()` on closure. Activating a directory
  bookmark closes the page and navigates the file tree; activating a file
  bookmark opens it with the configured opener and keeps the page open so the
  user can continue browsing.

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

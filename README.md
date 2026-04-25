# splorer

A keyboard- and mouse-driven terminal file explorer built in Go.

## What it does

splorer opens in your current working directory and displays a scrollable list
of files and directories. Directories are shown first, sorted
case-insensitively, followed by files in the same order. Each row shows the
entry name, size (or "dir"), and last-modified date.

The file list refreshes automatically every second when the contents of the
current directory change. If the directory itself is deleted, splorer navigates
to the nearest ancestor that still exists.

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
| `Ctrl+F` | Open Find → By Name directly (power-user shortcut) |
| `Ctrl+B` | Open the Create Bookmark dialog for the selected entry |
| `Alt+F` | Open the Find dropdown menu |
| `Alt+O` | Open the Openers dialog |
| `Alt+B` | Open the Bookmarks page |
| `Alt+S` | Open the Sort dialog |
| Click on a menu label | Open that menu's dialog (Find opens the dropdown) |

**Find dropdown** (opened with `Alt+F` or by clicking `Find ▾`)

| Input | Action |
|---|---|
| `↑` / `k` · `↓` / `j` | Move selection |
| `n` | Activate **By Name** |
| `c` | Activate **By Content** |
| `Enter` / `→` / `l` | Activate the highlighted option |
| Single click on an option | Activate it |
| `Esc` | Close the dropdown |

**Find / By Name view** (activated from the Find dropdown or `Ctrl+F`)

| Input | Action |
|---|---|
| Typing | Update the search pattern |
| `Alt+I` | Toggle case-insensitive matching (default: on) |
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

**Find / By Content view** (activated from the Find dropdown)

| Input | Action |
|---|---|
| Typing in **Pattern** | Update the search pattern |
| Paste (bracketed paste) | Insert clipboard content at the focused field's cursor; CR/LF are stripped |
| `Tab` / `Shift+Tab` | Switch focus between the Pattern and Ext fields |
| Typing in **Ext** | Comma-separated extension filter (e.g. `.go,.md`; empty = all files) |
| `Alt+R` | Toggle regex mode on/off (always active, regardless of focus) |
| `Alt+I` | Toggle case-insensitive matching (always active) |
| `Enter` | Start the search |
| `↑` / `k` · `↓` / `j` | Move cursor up/down in results |
| `PgUp` / `PgDn` | Scroll one page |
| `Enter` / `→` / `l` · Double-click | Open the file at that match |
| Single click on a result | Move cursor to that row |
| `Esc` | Cancel a running search / close the view |

Content search scans file contents line by line starting from the directory
the file tree is currently showing. Each result shows `relative/path:line:
matched text`, with the matched text dimmed so paths scan quickly. Matching is
a plain substring by default; toggle `Alt+R` to interpret the pattern as a
stdlib-regex (RE2 flavor — no lookarounds or backrefs). `Alt+I` makes the
match case-insensitive. The header reminds you of both toggles' current
state so they're discoverable without consulting this table.

Files that are probably binary are skipped (heuristic: the first 8 KB
contains a NUL byte — the same rule grep / git / ripgrep use). Files larger
than 10 MB and symbolic links are also skipped. If you use the extension
filter (`Ext: .go,.md`), matching files go through the binary check but
non-matching files are cheaply skipped before any bytes are read.

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

# Build the binary for the current OS / architecture
go build -o splorer .    # on Linux/macOS
go build -o splorer.exe .  # on Windows (or omit -o — `go build` picks the .exe suffix automatically)

# Run
./splorer
```

To start in a specific directory, `cd` there first or install the binary on
your `$PATH`:

```sh
go install github.com/bjcarnes/splorer@latest
```

### Cross-compiling

Go can produce a binary for any target OS from any host by setting the `GOOS`
environment variable before `go build`. splorer supports Linux and Windows;
`GOARCH=amd64` is the default on both and rarely needs to be set. The syntax
for setting env vars depends on your shell:

**bash / zsh** (one-shot, env var scoped to this command):

```sh
GOOS=linux   go build -o splorer      .   # Linux binary
GOOS=windows go build -o splorer.exe  .   # Windows binary
```

**PowerShell** (env var persists in the session until cleared):

```powershell
$env:GOOS="linux";   go build -o splorer     .
$env:GOOS="windows"; go build -o splorer.exe .
Remove-Item Env:GOOS   # return to native builds
```

**Windows cmd.exe**:

```bat
set GOOS=linux   && go build -o splorer     .
set GOOS=windows && go build -o splorer.exe .
set GOOS=
```

The resulting binaries are self-contained — the `start` / `xdg-open`
platform split is resolved at compile time via Go build tags in
`internal/opener`, so the `linux` binary never references `start`, and
vice versa.

## Leaving your shell in the last navigated directory

By default splorer exits back to whichever directory your shell was in when
you launched it — a child process can't change its parent shell's working
directory on its own. To have your shell land in the directory you were
viewing when you quit, install a small wrapper function for your shell.
splorer ships the wrapper itself; no extra files to distribute.

**PowerShell** — add this line to your `$PROFILE`:

```powershell
Invoke-Expression (& splorer.exe init powershell | Out-String)
```

**bash** — add this line to `~/.bashrc`:

```sh
eval "$(splorer init bash)"
```

**zsh** — add this line to `~/.zshrc`:

```sh
eval "$(splorer init zsh)"
```

Each shell's startup reads the wrapper back from `splorer init <shell>` and
defines a `splorer` function that shadows the bare binary name. When you
invoke `splorer`, the wrapper creates a temp file, runs the real binary with
`--cd-file <temp>`, and after exit reads the final directory out of the temp
file and `cd`s your shell there. Run `splorer init <shell>` on its own to see
the exact wrapper.

The `--cd-file <path>` flag is part of the public interface: splorer writes
the directory you last navigated to into that file on exit. Shells or scripts
other than the three above can use it directly.

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
    │   ├── app.go                Root tea.Model. Owns all sub-components, routes
    │   │                         messages, handles menu bar activation, dispatches
    │   │                         file-open and directory-navigate events. Exposes
    │   │                         CWD() for --cd-file; intercepts q/Esc to quit.
    │   └── app_test.go
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
    │   ├── menubar.go            MenuBar, Item, and SubItem types. Items are
    │   │                         activated by hotkey or click. Items with
    │   │                         SubItems emit OpenDropdownMsg instead of
    │   │                         their own Msg, letting the app mount a
    │   │                         finddropdown component below them.
    │   └── menubar_test.go
    ├── finddropdown/
    │   ├── dropdown.go           Small overlay dropdown component for menubar
    │   │                         items with sub-items. Handles arrow nav,
    │   │                         letter hotkeys (e.g. n, c), Enter, Esc, and
    │   │                         click hit-testing. app.View splices its
    │   │                         rendered box over the body using ANSI-aware
    │   │                         column manipulation.
    │   └── dropdown_test.go
    ├── contentsearch/
    │   ├── matcher.go            Mode (Exact / Regex), Options, matcher
    │   │                         interface, extension-filter parser.
    │   ├── matcher_test.go
    │   ├── walker.go             runContentSearch goroutine: filepath.WalkDir
    │   │                         with symlink skip, 10 MB size cap, NUL-byte
    │   │                         binary detection, bufio line scan, per-line
    │   │                         match dispatch.
    │   ├── walker_test.go
    │   └── model.go              Find-by-content view: two text fields
    │                             (Pattern, Ext), Tab focus cycling, Alt+R
    │                             regex toggle, Alt+I case toggle, streamed
    │                             result batches via channel, result list
    │                             navigation.
    ├── opener/
    │   ├── opener.go             OpenFileWith(path, prog) — shared across platforms.
    │   ├── opener_unix.go        OpenFile() via xdg-open          (build tag: !windows)
    │   ├── opener_windows.go     OpenFile() via cmd /c start "" … (build tag: windows)
    │   ├── opener_test.go
    │   └── opener_integration_test.go   (build tag: integration)
    ├── shellinit/
    │   ├── shellinit.go          Script(shell) returns the embedded wrapper
    │   │                         function for bash, zsh, or powershell. Used by
    │   │                         `splorer init <shell>` so the shell can cd into
    │   │                         the final --cd-file directory on exit.
    │   └── shellinit_test.go
    └── search/
        ├── model.go              Find-by-name view: filename-wildcard match,
        │                         text-input phase, background recursive walk
        │                         (filepath.WalkDir + context cancellation),
        │                         streaming result batches via channel,
        │                         sorted results list, keyboard/mouse nav.
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

- **Find is a menu-bar dropdown; sub-items carry their own messages.**  
  `menubar.Item` has a `SubItems []SubItem` field. When a user activates an
  item with sub-items (Alt+F or click), the menubar emits
  `menubar.OpenDropdownMsg{Index}` rather than the item's own `Msg`; the app
  responds by mounting a `finddropdown.Model` anchored to the menu item's
  column. The dropdown's activation cmd emits the chosen sub-item's message
  (`openSearchByNameMsg` / `openSearchByContentMsg`), and the app routes from
  there — no special-casing of Find inside the menubar package. Rendering the
  dropdown over the body uses `charmbracelet/x/ansi.Truncate` / `TruncateLeft`
  so ANSI-styled background cells outside the dropdown's column range survive
  the splice.

- **By-Content search skips binaries via the first-NUL-byte heuristic.**  
  `contentsearch.scanFile` reads up to 8 KB, scans for a NUL byte, and
  skips the file if one is found. This is the same rule grep, git, and
  ripgrep use, and it correctly classifies images, executables, compressed
  archives, and most other binaries without reading them end-to-end. The
  sample bytes are also the start of the line scan (via `io.MultiReader`),
  so we don't re-read the prefix. Files over 10 MB and all symbolic links
  are skipped outright. The extension filter applies before the binary
  check, so constraining to `.go,.md` means we never even open unrelated
  files.

- **Alt+R / Alt+I toggles instead of Tab-cycled form fields.**  
  Content search has a regex toggle and a case toggle in addition to two
  text fields (Pattern, Ext). Putting the toggles on `Alt+R` / `Alt+I` keeps
  them always-active regardless of which text field has focus, so the user
  never has to think about "where am I typing". Tab is reserved for
  switching between the two text fields — there's no way around needing
  some focus cycling with multiple text inputs.

- **The file tree is watched with a polling loop, not OS events.**
  `filetree.WatchCmd()` returns a one-shot Bubble Tea command that sleeps one
  second, reads the current directory, and emits `DirChangedMsg` (contents
  changed or transient error) or `DirGoneMsg` (directory removed). The handler
  in `filetree.Update` re-queues a fresh `WatchCmd` on every tick, forming a
  self-perpetuating chain. The sort order and directory path are baked into each
  tick so stale messages from a previous navigation or sort change are silently
  discarded instead of overwriting the current view. `app.Model.Init` starts the
  initial chain; every navigation that changes the CWD returns a new `WatchCmd`
  to replace it. `DirChangedMsg` and `DirGoneMsg` are caught in `app.Update`
  before the overlay-routing branches so the listing stays current even while a
  search or dialog is open.

- **`--cd-file` hands the exit directory back to the parent shell.**  
  A child process can't change its parent shell's cwd, so the usual "launcher
  leaves you in the last directory" trick requires shell cooperation. splorer
  takes the standard file-manager approach (same as `lf`, `nnn`, `yazi`): if
  `--cd-file <path>` is set, `main` writes `app.Model.CWD()` to that file on
  clean exit, and the shell wrapper function (emitted by `splorer init
  <shell>`) reads the file after splorer returns and runs `cd` on the user's
  behalf. The wrapper is shipped inside the binary as a string constant, so
  distribution is still a single file.

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

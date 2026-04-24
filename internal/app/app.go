package app

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bjcarnes/splorer/internal/associations"
	"github.com/bjcarnes/splorer/internal/bookmarks"
	"github.com/bjcarnes/splorer/internal/contentsearch"
	"github.com/bjcarnes/splorer/internal/filetree"
	"github.com/bjcarnes/splorer/internal/finddropdown"
	"github.com/bjcarnes/splorer/internal/menubar"
	"github.com/bjcarnes/splorer/internal/opener"
	"github.com/bjcarnes/splorer/internal/search"
	"github.com/bjcarnes/splorer/internal/sortdialog"
)

// menuBarHeight is the number of terminal lines the menu bar occupies.
const menuBarHeight = 1

// openOpenersMsg is dispatched when the user activates the Openers menu item.
type openOpenersMsg struct{}

// openSearchByNameMsg is dispatched when the user activates Find → By Name
// (or presses Ctrl+F as a shortcut to the same thing).
type openSearchByNameMsg struct{}

// openSearchByContentMsg is dispatched when the user activates Find → By Content.
type openSearchByContentMsg struct{}

// openBookmarksMsg is dispatched when the user activates the Bookmarks menu item.
type openBookmarksMsg struct{}

// openSortMsg is dispatched when the user activates the Sort menu item.
type openSortMsg struct{}

// Model is the root Bubble Tea model. It owns the menu bar, the file tree,
// and (when open) overlays for search, openers, bookmarks, or create-bookmark.
type Model struct {
	menu     menubar.MenuBar
	filetree filetree.Model

	dialog     associations.Dialog
	dialogOpen bool

	srch       search.Model
	searchOpen bool

	csrch      contentsearch.Model
	csrchOpen  bool

	findDropdown finddropdown.Model
	dropdownOpen bool

	bmarksPage  bookmarks.Page
	bmarksOpen  bool
	createBmark bookmarks.CreateDialog
	createOpen  bool

	sortDlg     sortdialog.Dialog
	sortDlgOpen bool

	assocs       map[string]string
	bookmarkList []bookmarks.Bookmark

	width  int
	height int
}

// New creates a root model starting in cwd.
func New(cwd string) Model {
	assocs, _ := associations.Load()
	bmarks, _ := bookmarks.Load()

	mb := menubar.New([]menubar.Item{
		{
			Label:  "Find",
			Hotkey: "alt+f",
			SubItems: []menubar.SubItem{
				{Label: "By Name", Key: 'n', Msg: openSearchByNameMsg{}},
				{Label: "By Content", Key: 'c', Msg: openSearchByContentMsg{}},
			},
		},
		{
			Label:  "Openers",
			Hotkey: "alt+o",
			Msg:    openOpenersMsg{},
		},
		{
			Label:  "Bookmarks",
			Hotkey: "alt+b",
			Msg:    openBookmarksMsg{},
		},
		{
			Label:  "Sort",
			Hotkey: "alt+s",
			Msg:    openSortMsg{},
		},
	})

	return Model{
		menu:         mb,
		filetree:     filetree.New(cwd),
		assocs:       assocs,
		bookmarkList: bmarks,
	}
}

// Init satisfies tea.Model; no initial commands are needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// CWD returns the directory the file tree is currently showing. Exposed so
// main.go can read the final navigated directory after Program.Run returns
// (used by the --cd-file flag).
func (m Model) CWD() string {
	return m.filetree.CWD()
}

// Update routes messages through the active component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Dropdown takes priority over other overlays — it only opens from the
	// main screen, so there's no ambiguity with deeper overlays.
	if m.dropdownOpen {
		return m.updateFindDropdown(msg)
	}

	if m.searchOpen {
		return m.updateSearch(msg)
	}
	if m.csrchOpen {
		return m.updateContentSearch(msg)
	}
	if m.dialogOpen {
		return m.updateDialog(msg)
	}
	if m.bmarksOpen {
		return m.updateBookmarks(msg)
	}
	if m.createOpen {
		return m.updateCreateBmark(msg)
	}
	if m.sortDlgOpen {
		return m.updateSortDialog(msg)
	}

	switch msg := msg.(type) {

	case menubar.OpenDropdownMsg:
		// Anchor the dropdown under the clicked/hotkeyed item by reading the
		// menu bar's item ranges.
		ranges := m.menu.ItemRanges()
		x := 1
		if msg.Index >= 0 && msg.Index < len(ranges) {
			x = ranges[msg.Index][0]
		}
		if msg.Index >= 0 && msg.Index < len(m.menu.Items) {
			m.findDropdown = finddropdown.New(m.menu.Items[msg.Index].SubItems, x)
			m.dropdownOpen = true
		}
		return m, nil

	case openSearchByNameMsg:
		m.srch = search.New(m.filetree.CWD(), m.width, m.height-menuBarHeight)
		m.searchOpen = true
		return m, nil

	case openSearchByContentMsg:
		m.csrch = contentsearch.New(m.filetree.CWD(), m.width, m.height-menuBarHeight)
		m.csrchOpen = true
		return m, nil

	case openOpenersMsg:
		m.dialog = associations.NewDialog(m.assocs)
		m.dialogOpen = true
		return m, nil

	case openBookmarksMsg:
		m.bmarksPage = bookmarks.NewPage(m.bookmarkList, m.width, m.height-menuBarHeight)
		m.bmarksOpen = true
		return m, nil

	case openSortMsg:
		m.sortDlg = sortdialog.New(m.filetree.CurrentSortOrder())
		m.sortDlgOpen = true
		return m, nil

	case bookmarks.NavigateDirMsg:
		if ft, err := m.filetree.NavigateTo(msg.Path); err == nil {
			m.filetree = ft
		}
		return m, nil

	case filetree.OpenFileMsg:
		m.openFile(msg.Path)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu.Width = msg.Width
		ft, cmd := m.filetree.Update(tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - menuBarHeight,
		})
		m.filetree = ft
		return m, cmd

	case tea.MouseClickMsg:
		if int(msg.Y) < menuBarHeight {
			if cmd := m.menu.HandleClick(int(msg.X)); cmd != nil {
				return m, cmd
			}
			return m, nil
		}
		ft, cmd := m.filetree.Update(tea.MouseClickMsg{
			X:      msg.X,
			Y:      msg.Y - menuBarHeight,
			Button: msg.Button,
			Mod:    msg.Mod,
		})
		m.filetree = ft
		return m, cmd

	case tea.MouseWheelMsg:
		ft, cmd := m.filetree.Update(msg)
		m.filetree = ft
		return m, cmd

	case tea.KeyPressMsg:
		// q and Esc quit the app, but only on the main screen — the overlay
		// branches above return before we get here, so overlays keep their own
		// Esc/typing semantics (e.g. Esc closes a dialog, "q" types into a
		// search field).
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		}
		// Ctrl+F is a power-user shortcut straight to the name-search view,
		// bypassing the Find dropdown. Keep it for continuity.
		if msg.String() == "ctrl+f" {
			m.srch = search.New(m.filetree.CWD(), m.width, m.height-menuBarHeight)
			m.searchOpen = true
			return m, nil
		}
		if msg.String() == "ctrl+b" {
			m.createBmark = bookmarks.NewCreateDialog(m.filetree.SelectedPath())
			m.createOpen = true
			return m, nil
		}
		if cmd := m.menu.HandleKey(msg); cmd != nil {
			return m, cmd
		}
		ft, cmd := m.filetree.Update(msg)
		m.filetree = ft
		return m, cmd
	}

	ft, cmd := m.filetree.Update(msg)
	m.filetree = ft
	return m, cmd
}

// updateFindDropdown routes events to the open dropdown. Clicks outside the
// dropdown bounds close it without being forwarded (so the underlying body
// isn't acted on). Activations produce the sub-item's Msg as a Cmd, which
// the event loop will feed back in on the next tick.
func (m Model) updateFindDropdown(msg tea.Msg) (tea.Model, tea.Cmd) {
	if click, ok := msg.(tea.MouseClickMsg); ok {
		if !m.findDropdown.Contains(int(click.X), int(click.Y)) {
			m.dropdownOpen = false
			return m, nil
		}
	}

	d, cmd := m.findDropdown.Update(msg)
	if d.IsClosed() {
		m.dropdownOpen = false
	} else {
		m.findDropdown = d
	}
	return m, cmd
}

// updateSearch routes events to the open name-search view.
func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.menu.Width = ws.Width
		ft, _ := m.filetree.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.filetree = ft
		sm, cmd := m.srch.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.srch = sm
		return m, cmd
	}

	switch msg := msg.(type) {

	case search.NavigateDirMsg:
		m.searchOpen = false
		if ft, err := m.filetree.NavigateTo(msg.Path); err == nil {
			m.filetree = ft
		}
		return m, nil

	case filetree.OpenFileMsg:
		m.openFile(msg.Path)
		return m, nil
	}

	sm, cmd := m.srch.Update(msg)
	m.srch = sm
	if sm.IsClosed() {
		m.searchOpen = false
	}
	return m, cmd
}

// updateContentSearch routes events to the open content-search view.
func (m Model) updateContentSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.menu.Width = ws.Width
		ft, _ := m.filetree.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.filetree = ft
		cs, cmd := m.csrch.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.csrch = cs
		return m, cmd
	}

	if fm, ok := msg.(filetree.OpenFileMsg); ok {
		m.openFile(fm.Path)
		return m, nil
	}

	cs, cmd := m.csrch.Update(msg)
	m.csrch = cs
	if cs.IsClosed() {
		m.csrchOpen = false
	}
	return m, cmd
}

// updateDialog routes an event to the open openers dialog and handles closure.
func (m Model) updateDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	d, cmd := m.dialog.Update(msg)
	if d.IsClosed() {
		m.dialogOpen = false
		m.assocs = d.Assocs()
		associations.Save(m.assocs) //nolint:errcheck
	} else {
		m.dialog = d
	}
	return m, cmd
}

// updateBookmarks routes events to the bookmarks page and handles closure/activation.
func (m Model) updateBookmarks(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.menu.Width = ws.Width
		ft, _ := m.filetree.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.filetree = ft
		p, cmd := m.bmarksPage.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.bmarksPage = p
		return m, cmd
	}

	switch msg := msg.(type) {
	case bookmarks.NavigateDirMsg:
		m.bmarksOpen = false
		m.bookmarkList = m.bmarksPage.Bookmarks()
		bookmarks.Save(m.bookmarkList) //nolint:errcheck
		if ft, err := m.filetree.NavigateTo(msg.Path); err == nil {
			m.filetree = ft
		}
		return m, nil

	case filetree.OpenFileMsg:
		m.openFile(msg.Path)
		return m, nil
	}

	p, cmd := m.bmarksPage.Update(msg)
	if p.IsClosed() {
		m.bmarksOpen = false
		m.bookmarkList = p.Bookmarks()
		bookmarks.Save(m.bookmarkList) //nolint:errcheck
	} else {
		m.bmarksPage = p
	}
	return m, cmd
}

// updateCreateBmark routes events to the create-bookmark dialog and handles closure.
func (m Model) updateCreateBmark(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.menu.Width = ws.Width
		ft, _ := m.filetree.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.filetree = ft
		return m, nil
	}

	d, cmd := m.createBmark.Update(msg)
	if d.IsClosed() {
		m.createOpen = false
		if d.IsSaved() {
			bm := bookmarks.Bookmark{
				Name: strings.TrimSpace(d.Name()),
				Path: d.BookmarkPath(),
			}
			m.bookmarkList = append(m.bookmarkList, bm)
			bookmarks.Save(m.bookmarkList) //nolint:errcheck
		}
	} else {
		m.createBmark = d
	}
	return m, cmd
}

// updateSortDialog routes events to the sort picker and applies a new sort order on save.
func (m Model) updateSortDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.menu.Width = ws.Width
		ft, _ := m.filetree.Update(tea.WindowSizeMsg{
			Width:  ws.Width,
			Height: ws.Height - menuBarHeight,
		})
		m.filetree = ft
		return m, nil
	}

	d, cmd := m.sortDlg.Update(msg)
	if d.IsClosed() {
		m.sortDlgOpen = false
		if d.IsSaved() {
			m.filetree = m.filetree.SetSortOrder(d.Chosen())
		}
	} else {
		m.sortDlg = d
	}
	return m, cmd
}

// openFile opens the file at path using the configured association or the
// platform default opener.
func (m Model) openFile(path string) {
	ext := strings.ToLower(filepath.Ext(path))
	if prog, ok := m.assocs[ext]; ok {
		opener.OpenFileWith(path, prog) //nolint:errcheck
	} else {
		opener.OpenFile(path) //nolint:errcheck
	}
}

// View renders the full screen.
func (m Model) View() tea.View {
	menuLine := m.menu.Render()

	var body string
	switch {
	case m.searchOpen:
		body = m.srch.Render()
	case m.csrchOpen:
		body = m.csrch.Render()
	case m.dialogOpen:
		body = m.dialog.Render(m.width, m.height-menuBarHeight)
	case m.bmarksOpen:
		body = m.bmarksPage.Render()
	case m.createOpen:
		body = m.createBmark.Render(m.width, m.height-menuBarHeight)
	case m.sortDlgOpen:
		body = m.sortDlg.Render(m.width, m.height-menuBarHeight)
	default:
		body = m.filetree.Render()
	}

	// The dropdown overlay, if open, is spliced onto the body rows so the
	// underlying filetree stays visible around it.
	if m.dropdownOpen {
		body = spliceOverlay(
			body,
			m.findDropdown.Render(),
			m.findDropdown.X(),
			m.findDropdown.Y()-menuBarHeight,
		)
	}

	v := tea.NewView(menuLine + "\n" + body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// spliceOverlay renders `over` onto `base` at (x, y), where y is 0-indexed
// from the first line of `base`. ANSI escapes in `base` are preserved
// outside the overlay region; inside the region, base content is replaced by
// over. Used to draw the Find dropdown on top of the file-tree body.
func spliceOverlay(base, over string, x, y int) string {
	baseLines := strings.Split(base, "\n")
	overLines := strings.Split(over, "\n")

	for i, line := range overLines {
		row := y + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = spliceAtColumn(baseLines[row], line, x)
	}
	return strings.Join(baseLines, "\n")
}

// spliceAtColumn returns base with its cells at [x, x+width(over)) replaced
// by over. Uses the ANSI-aware Truncate/TruncateLeft helpers from
// charmbracelet/x/ansi so styled base content outside the replaced region
// keeps its escape codes intact.
func spliceAtColumn(base, over string, x int) string {
	overW := ansi.StringWidth(over)
	baseW := ansi.StringWidth(base)

	left := ansi.Truncate(base, x, "")
	leftW := ansi.StringWidth(left)
	if leftW < x {
		left += strings.Repeat(" ", x-leftW)
	}

	var right string
	if x+overW < baseW {
		right = ansi.TruncateLeft(base, x+overW, "")
	}
	return left + over + right
}

package app

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/associations"
	"github.com/bjcarnes/splorer/internal/bookmarks"
	"github.com/bjcarnes/splorer/internal/filetree"
	"github.com/bjcarnes/splorer/internal/menubar"
	"github.com/bjcarnes/splorer/internal/opener"
	"github.com/bjcarnes/splorer/internal/search"
	"github.com/bjcarnes/splorer/internal/sortdialog"
)

// menuBarHeight is the number of terminal lines the menu bar occupies.
const menuBarHeight = 1

// openOpenersMsg is dispatched when the user activates the Openers menu item.
type openOpenersMsg struct{}

// openSearchMsg is dispatched when the user activates the Find menu item or
// presses the Ctrl+F shortcut.
type openSearchMsg struct{}

// openBookmarksMsg is dispatched when the user activates the Bookmarks menu item.
type openBookmarksMsg struct{}

// openSortMsg is dispatched when the user activates the Sort menu item.
type openSortMsg struct{}

// Model is the root Bubble Tea model. It owns the menu bar, the file tree,
// and (when open) overlays for search, openers, bookmarks, or create-bookmark.
type Model struct {
	menu       menubar.MenuBar
	filetree   filetree.Model
	dialog     associations.Dialog
	dialogOpen bool
	srch       search.Model
	searchOpen bool

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
			Msg:    openSearchMsg{},
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

// Update routes messages through the active component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.searchOpen {
		return m.updateSearch(msg)
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

	case openSearchMsg:
		m.srch = search.New(m.filetree.CWD(), m.width, m.height-menuBarHeight)
		m.searchOpen = true
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

// updateSearch routes events to the open search view.
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

// openFile opens the file at path using the configured association or xdg-open.
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

	v := tea.NewView(menuLine + "\n" + body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

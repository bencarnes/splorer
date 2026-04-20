package app

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/associations"
	"github.com/bjcarnes/splorer/internal/filetree"
	"github.com/bjcarnes/splorer/internal/menubar"
	"github.com/bjcarnes/splorer/internal/opener"
	"github.com/bjcarnes/splorer/internal/search"
)

// menuBarHeight is the number of terminal lines the menu bar occupies.
const menuBarHeight = 1

// openOpenersMsg is dispatched when the user activates the Openers menu item.
type openOpenersMsg struct{}

// openSearchMsg is dispatched when the user activates the Find menu item or
// presses the Ctrl+F shortcut.
type openSearchMsg struct{}

// Model is the root Bubble Tea model. It owns the menu bar, the file tree,
// and (when open) either the Openers dialog or the Find search view.
type Model struct {
	menu       menubar.MenuBar
	filetree   filetree.Model
	dialog     associations.Dialog
	dialogOpen bool
	srch       search.Model
	searchOpen bool
	assocs     map[string]string
	width      int
	height     int
}

// New creates a root model starting in cwd.
func New(cwd string) Model {
	assocs, _ := associations.Load() // best-effort; errors silently ignored at startup

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
	})

	return Model{
		menu:     mb,
		filetree: filetree.New(cwd),
		assocs:   assocs,
	}
}

// Init satisfies tea.Model; no initial commands are needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update routes messages through the active component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Search view takes exclusive control when open.
	if m.searchOpen {
		return m.updateSearch(msg)
	}

	// Dialog takes exclusive control when open.
	if m.dialogOpen {
		return m.updateDialog(msg)
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

	case filetree.OpenFileMsg:
		ext := strings.ToLower(filepath.Ext(msg.Path))
		if prog, ok := m.assocs[ext]; ok {
			opener.OpenFileWith(msg.Path, prog) //nolint:errcheck
		} else {
			opener.OpenFile(msg.Path) //nolint:errcheck
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu.Width = msg.Width
		// Give the filetree the height minus the menu bar row.
		ft, cmd := m.filetree.Update(tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - menuBarHeight,
		})
		m.filetree = ft
		return m, cmd

	case tea.MouseClickMsg:
		if int(msg.Y) < menuBarHeight {
			// Click landed on the menu bar.
			if cmd := m.menu.HandleClick(int(msg.X)); cmd != nil {
				return m, cmd
			}
			return m, nil
		}
		// Translate Y into the filetree's coordinate space.
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
		// Ctrl+F opens search regardless of other bindings.
		if msg.String() == "ctrl+f" {
			m.srch = search.New(m.filetree.CWD(), m.width, m.height-menuBarHeight)
			m.searchOpen = true
			return m, nil
		}
		// Menu bar hotkeys take priority over filetree bindings.
		if cmd := m.menu.HandleKey(msg); cmd != nil {
			return m, cmd
		}
		ft, cmd := m.filetree.Update(msg)
		m.filetree = ft
		return m, cmd
	}

	// Forward any unhandled messages (e.g. tea.QuitMsg) to the filetree.
	ft, cmd := m.filetree.Update(msg)
	m.filetree = ft
	return m, cmd
}

// updateSearch routes events to the open search view. It also intercepts
// messages that search emits as commands (NavigateDirMsg, OpenFileMsg) so the
// app can act on them while the search view is still "active".
func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Window resize must update both the filetree (kept in sync even when
	// hidden) and the search view.
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

	// Commands returned by the search model fire on subsequent Update calls.
	// Handle the two result-activation message types before they reach the
	// search model (which would ignore them anyway).
	switch msg := msg.(type) {

	case search.NavigateDirMsg:
		// Navigate the underlying file tree and close the search view.
		m.searchOpen = false
		if ft, err := m.filetree.NavigateTo(msg.Path); err == nil {
			m.filetree = ft
		}
		return m, nil

	case filetree.OpenFileMsg:
		// Open the file using the configured opener; leave search view open so
		// the user can continue browsing results.
		ext := strings.ToLower(filepath.Ext(msg.Path))
		if prog, ok := m.assocs[ext]; ok {
			opener.OpenFileWith(msg.Path, prog) //nolint:errcheck
		} else {
			opener.OpenFile(msg.Path) //nolint:errcheck
		}
		return m, nil
	}

	// All other messages (key events, mouse events, resultBatchMsg, …) go to
	// the search model.
	sm, cmd := m.srch.Update(msg)
	m.srch = sm

	if sm.IsClosed() {
		m.searchOpen = false
	}

	return m, cmd
}

// updateDialog routes an event to the open dialog and handles its closure.
func (m Model) updateDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	d, cmd := m.dialog.Update(msg)
	if d.IsClosed() {
		m.dialogOpen = false
		m.assocs = d.Assocs()
		associations.Save(m.assocs) //nolint:errcheck // best-effort save
	} else {
		m.dialog = d
	}
	return m, cmd
}

// View renders the full screen. AltScreen and mouse mode are set here so they
// are declared on every frame.
func (m Model) View() tea.View {
	menuLine := m.menu.Render()

	var body string
	switch {
	case m.searchOpen:
		body = m.srch.Render()
	case m.dialogOpen:
		body = m.dialog.Render(m.width, m.height-menuBarHeight)
	default:
		body = m.filetree.Render()
	}

	v := tea.NewView(menuLine + "\n" + body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

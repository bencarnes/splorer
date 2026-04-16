package app

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/associations"
	"github.com/bjcarnes/splorer/internal/filetree"
	"github.com/bjcarnes/splorer/internal/menubar"
	"github.com/bjcarnes/splorer/internal/opener"
)

// menuBarHeight is the number of terminal lines the menu bar occupies.
const menuBarHeight = 1

// openOpenersMsg is dispatched when the user activates the Openers menu item.
type openOpenersMsg struct{}

// Model is the root Bubble Tea model. It owns the menu bar, the file tree,
// and (when open) the Openers dialog.
type Model struct {
	menu       menubar.MenuBar
	filetree   filetree.Model
	dialog     associations.Dialog
	dialogOpen bool
	assocs     map[string]string
	width      int
	height     int
}

// New creates a root model starting in cwd.
func New(cwd string) Model {
	assocs, _ := associations.Load() // best-effort; errors silently ignored at startup

	mb := menubar.New([]menubar.Item{
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
	// While the dialog is open it receives all events exclusively.
	if m.dialogOpen {
		return m.updateDialog(msg)
	}

	switch msg := msg.(type) {

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
	if m.dialogOpen {
		body = m.dialog.Render(m.width, m.height-menuBarHeight)
	} else {
		body = m.filetree.Render()
	}

	v := tea.NewView(menuLine + "\n" + body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

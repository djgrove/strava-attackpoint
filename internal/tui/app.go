package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenMenu screen = iota
	screenSetup
	screenSync
)

type model struct {
	screen     screen
	menuCursor int
	setup      setupModel
	sync       syncModel
	quitting   bool
}

var menuItems = []string{"Setup Strava", "Sync Activities", "Quit"}

func NewModel() model {
	return model{
		screen: screenMenu,
		setup:  newSetupModel(),
		sync:   newSyncModel(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global quit handling.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.screen != screenMenu {
				m.screen = screenMenu
				return m, nil
			}
		}
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenSetup:
		var cmd tea.Cmd
		m.setup, cmd = m.setup.Update(msg)
		return m, cmd
	case screenSync:
		var cmd tea.Cmd
		m.sync, cmd = m.sync.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < len(menuItems)-1 {
				m.menuCursor++
			}
		case "enter":
			switch m.menuCursor {
			case 0:
				m.screen = screenSetup
				m.setup = newSetupModel()
				return m, m.setup.Init()
			case 1:
				m.screen = screenSync
				m.sync = newSyncModel()
				return m, m.sync.Init()
			case 2:
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenSetup:
		return m.viewWithNav(m.setup.View())
	case screenSync:
		return m.viewWithNav(m.sync.View())
	}
	return ""
}

func (m model) viewMenu() string {
	s := titleStyle.Render("strava-ap") + "\n"
	s += subtitleStyle.Render("Sync Strava activities to AttackPoint.org") + "\n\n"

	for i, item := range menuItems {
		if i == m.menuCursor {
			s += selectedItemStyle.Render("> " + item) + "\n"
		} else {
			s += menuItemStyle.Render("  " + item) + "\n"
		}
	}

	s += helpStyle.Render("\nj/k or arrows to navigate, enter to select, ctrl+c to quit")
	return s
}

func (m model) viewWithNav(content string) string {
	return content + helpStyle.Render("\nesc to go back, ctrl+c to quit")
}

// Run starts the TUI application.
func Run() error {
	p := tea.NewProgram(NewModel())
	_, err := p.Run()
	return err
}

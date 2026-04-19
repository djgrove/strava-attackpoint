package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/djgrove/strava-attackpoint/internal/strava"
)

type setupStep int

const (
	setupStepConfirm setupStep = iota
	setupStepOAuth
	setupStepDone
)

type setupModel struct {
	step   setupStep
	status string
	err    error
}

type oauthCompleteMsg struct{ err error }

func newSetupModel() setupModel {
	return setupModel{
		step: setupStepConfirm,
	}
}

func (m setupModel) Init() tea.Cmd {
	return nil
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" && m.step == setupStepConfirm {
			m.step = setupStepOAuth
			m.status = "Opening browser for Strava authorization..."
			return m, m.startOAuth()
		}

	case oauthCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = fmt.Sprintf("Setup failed: %v", msg.err)
		} else {
			m.step = setupStepDone
			m.status = "Setup complete! Tokens saved securely to your keychain."
		}
		return m, nil
	}

	return m, nil
}

func (m setupModel) View() string {
	s := titleStyle.Render("Strava Setup") + "\n\n"

	switch m.step {
	case setupStepConfirm:
		s += "Press Enter to open your browser and authorize with Strava.\n"
	case setupStepOAuth:
		s += successStyle.Render(m.status) + "\n"
	case setupStepDone:
		if m.err != nil {
			s += errorStyle.Render(m.status) + "\n"
		} else {
			s += successStyle.Render(m.status) + "\n"
		}
	}

	return s
}

func (m setupModel) startOAuth() tea.Cmd {
	return func() tea.Msg {
		err := strava.RunOAuthFlow()
		return oauthCompleteMsg{err: err}
	}
}

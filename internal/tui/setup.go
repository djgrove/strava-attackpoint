package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/djgrove/strava-attackpoint/internal/strava"
)

type setupStep int

const (
	setupStepClientID setupStep = iota
	setupStepClientSecret
	setupStepOAuth
	setupStepDone
)

type setupModel struct {
	step         setupStep
	clientID     textinput.Model
	clientSecret textinput.Model
	status       string
	err          error
}

type oauthCompleteMsg struct{ err error }

func newSetupModel() setupModel {
	ci := textinput.New()
	ci.Placeholder = "e.g., 200536"
	ci.Focus()

	cs := textinput.New()
	cs.Placeholder = "your client secret"
	cs.EchoMode = textinput.EchoPassword

	return setupModel{
		step:         setupStepClientID,
		clientID:     ci,
		clientSecret: cs,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			switch m.step {
			case setupStepClientID:
				if m.clientID.Value() != "" {
					m.step = setupStepClientSecret
					m.clientID.Blur()
					m.clientSecret.Focus()
					return m, textinput.Blink
				}
			case setupStepClientSecret:
				if m.clientSecret.Value() != "" {
					m.step = setupStepOAuth
					m.clientSecret.Blur()
					m.status = "Opening browser for Strava authorization..."
					return m, m.startOAuth()
				}
			}
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

	// Update active text input.
	var cmd tea.Cmd
	switch m.step {
	case setupStepClientID:
		m.clientID, cmd = m.clientID.Update(msg)
	case setupStepClientSecret:
		m.clientSecret, cmd = m.clientSecret.Update(msg)
	}
	return m, cmd
}

func (m setupModel) View() string {
	s := titleStyle.Render("Strava API Setup") + "\n"
	s += subtitleStyle.Render("Create an app at https://www.strava.com/settings/api") + "\n"
	s += subtitleStyle.Render("Set 'Authorization Callback Domain' to: localhost") + "\n\n"

	switch m.step {
	case setupStepClientID:
		s += "Client ID:\n"
		s += m.clientID.View() + "\n"
	case setupStepClientSecret:
		s += "Client ID: " + successStyle.Render(m.clientID.Value()) + "\n\n"
		s += "Client Secret:\n"
		s += m.clientSecret.View() + "\n"
	case setupStepOAuth, setupStepDone:
		s += "Client ID: " + successStyle.Render(m.clientID.Value()) + "\n"
		s += "Client Secret: " + successStyle.Render("********") + "\n\n"
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
		err := strava.RunOAuthFlow(m.clientID.Value(), m.clientSecret.Value())
		return oauthCompleteMsg{err: err}
	}
}

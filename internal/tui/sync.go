package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/djgrove/strava-attackpoint/internal/config"
	"github.com/djgrove/strava-attackpoint/internal/strava"
	syncpkg "github.com/djgrove/strava-attackpoint/internal/sync"
	"time"
)

type syncStep int

const (
	syncStepDate syncStep = iota
	syncStepAPUsername
	syncStepAPPassword
	syncStepRunning
	syncStepDone
)

type syncModel struct {
	step       syncStep
	dateInput  textinput.Model
	username   textinput.Model
	password   textinput.Model
	spinner    spinner.Model
	results    []syncpkg.Result
	status     string
	err        error
}

type syncCompleteMsg struct {
	results []syncpkg.Result
	err     error
}

func newSyncModel() syncModel {
	di := textinput.New()
	di.Placeholder = "YYYY-MM-DD"
	di.Focus()

	un := textinput.New()
	un.Placeholder = "username"

	pw := textinput.New()
	pw.Placeholder = "password"
	pw.EchoMode = textinput.EchoPassword

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return syncModel{
		step:      syncStepDate,
		dateInput: di,
		username:  un,
		password:  pw,
		spinner:   sp,
	}
}

func (m syncModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m syncModel) Update(msg tea.Msg) (syncModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			switch m.step {
			case syncStepDate:
				if m.dateInput.Value() != "" {
					_, err := time.Parse("2006-01-02", m.dateInput.Value())
					if err != nil {
						m.status = "Invalid date format. Use YYYY-MM-DD."
						return m, nil
					}
					m.step = syncStepAPUsername
					m.dateInput.Blur()
					m.username.Focus()
					m.status = ""
					return m, textinput.Blink
				}
			case syncStepAPUsername:
				if m.username.Value() != "" {
					m.step = syncStepAPPassword
					m.username.Blur()
					m.password.Focus()
					return m, textinput.Blink
				}
			case syncStepAPPassword:
				if m.password.Value() != "" {
					m.step = syncStepRunning
					m.password.Blur()
					m.status = "Syncing..."
					return m, tea.Batch(m.spinner.Tick, m.startSync())
				}
			}
		}

	case spinner.TickMsg:
		if m.step == syncStepRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case syncCompleteMsg:
		m.step = syncStepDone
		m.results = msg.results
		m.err = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Sync failed: %v", msg.err)
		} else {
			synced, skipped, failed := 0, 0, 0
			for _, r := range msg.results {
				switch r.Status {
				case "synced":
					synced++
				case "skipped":
					skipped++
				case "failed":
					failed++
				}
			}
			m.status = fmt.Sprintf("Done! %d synced, %d skipped, %d failed", synced, skipped, failed)
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case syncStepDate:
		m.dateInput, cmd = m.dateInput.Update(msg)
	case syncStepAPUsername:
		m.username, cmd = m.username.Update(msg)
	case syncStepAPPassword:
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}

func (m syncModel) View() string {
	s := titleStyle.Render("Sync Activities") + "\n\n"

	switch m.step {
	case syncStepDate:
		s += "Sync activities since:\n"
		s += m.dateInput.View() + "\n"
		if m.status != "" {
			s += warningStyle.Render(m.status) + "\n"
		}
	case syncStepAPUsername:
		s += "Since: " + successStyle.Render(m.dateInput.Value()) + "\n\n"
		s += "AttackPoint username:\n"
		s += m.username.View() + "\n"
	case syncStepAPPassword:
		s += "Since: " + successStyle.Render(m.dateInput.Value()) + "\n"
		s += "AP user: " + successStyle.Render(m.username.Value()) + "\n\n"
		s += "AttackPoint password:\n"
		s += m.password.View() + "\n"
	case syncStepRunning:
		s += m.spinner.View() + " " + m.status + "\n"
	case syncStepDone:
		if m.err != nil {
			s += errorStyle.Render(m.status) + "\n"
		} else {
			s += successStyle.Render(m.status) + "\n"
		}
		// Show failed activities.
		for _, r := range m.results {
			if r.Status == "failed" {
				s += errorStyle.Render(fmt.Sprintf("  FAILED: %s — %v", r.ActivityName, r.Error)) + "\n"
			}
		}
	}

	return s
}

func (m syncModel) startSync() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.LoadConfig()
		if err != nil {
			return syncCompleteMsg{err: fmt.Errorf("loading config: %w", err)}
		}
		token, _ := config.GetAccessToken()
		if token == "" {
			return syncCompleteMsg{err: fmt.Errorf("Strava not configured — use Setup first")}
		}

		state, err := config.LoadSyncState()
		if err != nil {
			return syncCompleteMsg{err: fmt.Errorf("loading sync state: %w", err)}
		}

		stravaClient, err := strava.NewClient(cfg)
		if err != nil {
			return syncCompleteMsg{err: err}
		}

		apClient, err := attackpoint.NewClient()
		if err != nil {
			return syncCompleteMsg{err: err}
		}
		if err := apClient.Login(m.username.Value(), m.password.Value()); err != nil {
			return syncCompleteMsg{err: fmt.Errorf("AP login failed: %w", err)}
		}

		since, _ := time.Parse("2006-01-02", m.dateInput.Value())
		engine := syncpkg.NewEngine(stravaClient, apClient, state)
		results, err := engine.SyncSince(since, time.Time{})
		return syncCompleteMsg{results: results, err: err}
	}
}

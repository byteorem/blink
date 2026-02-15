package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type SyncFileMsg struct{}
type SyncDoneMsg struct{ Count int }

type SyncModel struct {
	total    int
	copied   int
	progress progress.Model
	done     bool
	count    int
}

func NewSyncModel(total int) SyncModel {
	p := progress.New(progress.WithDefaultGradient())
	return SyncModel{
		total:    total,
		progress: p,
	}
}

func (m SyncModel) Init() tea.Cmd {
	return nil
}

func (m SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case SyncFileMsg:
		m.copied++
		if m.copied >= m.total {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case SyncDoneMsg:
		m.done = true
		m.count = msg.Count
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil
	}

	return m, nil
}

func (m SyncModel) View() string {
	if m.done {
		return ""
	}

	pct := 0.0
	if m.total > 0 {
		pct = float64(m.copied) / float64(m.total)
	}

	s := "\n"
	s += " " + headerStyle.Render("✨ blink") + "\n\n"
	s += " " + m.progress.ViewAs(pct) + "\n\n"
	s += fmt.Sprintf("  Syncing files... %d/%d\n", m.copied, m.total)
	return s
}

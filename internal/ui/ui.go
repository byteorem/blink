// Package ui provides Bubbletea TUI components for blink.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/byteorem/blink/internal/copier"
	"github.com/byteorem/blink/internal/watcher"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxChangelog = 5

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")) // yellow/gold
	dotStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))             // green
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))            // dim label
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))             // cyan
	arrowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))            // dim arrow
	copiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))             // green
	removedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))              // red
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type changeEntry struct {
	time    time.Time
	relPath string
	action  string
}

// Model is the Bubbletea model for the main watcher TUI.
type Model struct {
	addonName  string
	targetPath string
	fileCount  int
	spinner    spinner.Model
	changelog  []changeEntry
	srcDir     string
	dstDir     string
	eventCh    <-chan watcher.Event
	quitting   bool
}

// WatcherEventMsg wraps a watcher event for the Bubbletea update loop.
type WatcherEventMsg watcher.Event

// FileChangedMsg signals that a file was synced or removed.
type FileChangedMsg struct {
	relPath string
	action  string
}

// NewModel creates a new watcher TUI model.
func NewModel(addonName, targetPath, srcDir, dstDir string, fileCount int, eventCh <-chan watcher.Event) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	return Model{
		addonName:  addonName,
		targetPath: targetPath,
		fileCount:  fileCount,
		spinner:    s,
		srcDir:     srcDir,
		dstDir:     dstDir,
		eventCh:    eventCh,
	}
}

// Init starts the spinner and watcher listener.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, listenToWatcher(m.eventCh))
}

func listenToWatcher(ch <-chan watcher.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return tea.Quit()
		}
		return WatcherEventMsg(ev)
	}
}

// Update handles incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case WatcherEventMsg:
		ev := watcher.Event(msg)
		return m, tea.Batch(
			m.handleEvent(ev),
			listenToWatcher(m.eventCh),
		)

	case FileChangedMsg:
		m.fileCount++
		entry := changeEntry{
			time:    time.Now(),
			relPath: msg.relPath,
			action:  msg.action,
		}
		m.changelog = append(m.changelog, entry)
		if len(m.changelog) > maxChangelog {
			m.changelog = m.changelog[len(m.changelog)-maxChangelog:]
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleEvent(ev watcher.Event) tea.Cmd {
	return func() tea.Msg {
		dstPath := filepath.Join(m.dstDir, ev.RelPath)
		srcPath := filepath.Join(m.srcDir, ev.RelPath)

		switch ev.Op {
		case watcher.OpRemove:
			_ = copier.DeleteFile(dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "removed"}
		case watcher.OpRename:
			if _, err := os.Stat(srcPath); err == nil {
				_ = copier.CopyFile(srcPath, dstPath)
				return FileChangedMsg{relPath: ev.RelPath, action: "copied"}
			}
			_ = copier.DeleteFile(dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "removed"}
		default:
			_ = copier.CopyFile(srcPath, dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "copied"}
		}
	}
}

// View renders the TUI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	s := "\n"
	s += " " + headerStyle.Render("✨ blink") + "\n\n"
	s += dotStyle.Render(" ●") + labelStyle.Render(" Watching   ") + m.addonName + "\n"
	s += dotStyle.Render(" ●") + labelStyle.Render(" Target     ") + m.targetPath + "\n"
	s += dotStyle.Render(" ●") + labelStyle.Render(" Files      ") + fmt.Sprintf("%d synced", m.fileCount) + "\n"
	s += "\n"
	s += " " + m.spinner.View() + " Watching for changes...\n"
	s += "\n"

	for _, entry := range m.changelog {
		ts := entry.time.Format("15:04:05")
		actionStyled := entry.action
		switch entry.action {
		case "copied":
			actionStyled = copiedStyle.Render(entry.action)
		case "removed":
			actionStyled = removedStyle.Render(entry.action)
		}
		s += dimStyle.Render("  "+ts) + "  " + pathStyle.Render(entry.relPath) + " " + arrowStyle.Render("→") + " " + actionStyled + "\n"
	}

	if len(m.changelog) > 0 {
		s += "\n"
	}

	s += dimStyle.Render("  Press q to quit") + "\n"
	return s
}

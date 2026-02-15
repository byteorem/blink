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

type WatcherEventMsg watcher.Event
type FileChangedMsg struct {
	relPath string
	action  string
}

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
			copier.DeleteFile(dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "removed"}
		case watcher.OpRename:
			if _, err := os.Stat(srcPath); err == nil {
				copier.CopyFile(srcPath, dstPath)
				return FileChangedMsg{relPath: ev.RelPath, action: "copied"}
			}
			copier.DeleteFile(dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "removed"}
		default:
			copier.CopyFile(srcPath, dstPath)
			return FileChangedMsg{relPath: ev.RelPath, action: "copied"}
		}
	}
}

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

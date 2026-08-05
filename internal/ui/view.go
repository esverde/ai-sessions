package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/esverde/ais/internal/provider"
)

type sessionDelegate struct {
	title lipgloss.Style
	desc  lipgloss.Style
	meta  lipgloss.Style
}

func newSessionDelegate() sessionDelegate {
	return sessionDelegate{
		title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DCFFF")),
		desc:  lipgloss.NewStyle().Foreground(lipgloss.Color("#A9B1D6")),
		meta:  lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
	}
}

func (d sessionDelegate) Height() int                         { return 3 }
func (d sessionDelegate) Spacing() int                        { return 0 }
func (d sessionDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(writer io.Writer, model list.Model, index int, item list.Item) {
	value, ok := item.(sessionItem)
	if !ok {
		return
	}
	selected := index == model.Index()
	marker := "  "
	if selected {
		marker = "▸ "
	}
	titleStyle := d.title
	if selected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#BB9AF7"))
	}
	line1 := fmt.Sprintf("%s[%s] %s", marker, value.value.Provider, provider.Truncate(value.value.Title, 72))
	line2 := fmt.Sprintf("  %s · %s", formatTime(value.value.UpdatedAt), provider.Truncate(value.value.Cwd, 100))
	preview := value.value.LastUser
	if preview == "" {
		preview = value.value.LastAssistant
	}
	line3 := "  " + provider.Truncate(preview, 120)
	if preview == "" {
		line3 = "  (no text preview)"
	}
	_, _ = io.WriteString(writer, titleStyle.Render(line1)+"\n"+d.desc.Render(line2)+"\n"+d.meta.Render(line3))
}

func (m Model) View() tea.View {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7AA2F7")).Render("ais")
	scope := "current directory"
	if m.scopeAll {
		scope = "all projects"
	}
	header += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Render("["+m.provider+"]")
	header += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")).Render(scope)
	header += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("sort:"+m.sortMode)

	footer := []string{m.status}
	if m.configPath != "" {
		footer = append(footer, "config: "+m.configPath)
	}
	footer = append(footer, "Enter resume · / search · ? help · q quit")

	content := strings.Join([]string{header, m.list.View(), lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(strings.Join(footer, "\n"))}, "\n")
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "ais - AI sessions"
	return view
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown time"
	}
	return value.Local().Format("2006-01-02 15:04")
}

var _ list.ItemDelegate = sessionDelegate{}
var _ list.Item = sessionItem{}

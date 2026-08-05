package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/esverde/ais/internal/config"
	"github.com/esverde/ais/internal/launcher"
	"github.com/esverde/ais/internal/provider"
	"github.com/esverde/ais/internal/session"
)

type sessionItem struct {
	value session.Session
}

func (item sessionItem) FilterValue() string {
	return strings.Join([]string{item.value.Provider, item.value.ID, item.value.Cwd, item.value.Title, item.value.LastUser, item.value.LastAssistant}, " ")
}

type scanFinishedMsg struct {
	sessions []session.Session
	errors   []error
}

type resumeFinishedMsg struct{ err error }

type Model struct {
	list       list.Model
	items      []session.Session
	config     config.Config
	configPath string
	currentDir string
	provider   string
	scopeAll   bool
	sortMode   string
	status     string
	lastScan   time.Time
	delegate   sessionDelegate
}

func NewModel(cfg config.Config, configPath, currentDir string) Model {
	delegate := newSessionDelegate()
	items := make([]list.Item, 0)
	component := list.New(items, delegate, 80, 20)
	component.SetShowTitle(false)
	component.SetShowStatusBar(false)
	component.SetShowPagination(true)
	component.SetShowHelp(false)
	component.SetShowFilter(true)
	component.DisableQuitKeybindings()
	return Model{
		list:       component,
		config:     cfg.Normalize(),
		configPath: configPath,
		currentDir: currentDir,
		provider:   cfg.Provider,
		scopeAll:   cfg.Scope == config.ScopeAll,
		sortMode:   cfg.Sort,
		status:     "Scanning session files…",
		delegate:   delegate,
	}
}

func (m Model) Init() tea.Cmd {
	return m.scan()
}

func (m Model) scan() tea.Cmd {
	options := session.ScanOptions{
		ScopeRoot:       m.currentDir,
		ScopeAll:        m.scopeAll,
		IncludeArchived: m.config.IncludeArchived,
		PreviewLength:   m.config.PreviewLength,
	}
	providerName := m.provider
	return func() tea.Msg {
		items, errors := provider.Discover(providerName, options)
		items = provider.Deduplicate(items)
		provider.Sort(items, m.sortMode)
		if m.config.MaxSessions > 0 && len(items) > m.config.MaxSessions {
			items = items[:m.config.MaxSessions]
		}
		return scanFinishedMsg{sessions: items, errors: errors}
	}
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(5, msg.Height-5))
		return m, nil
	case scanFinishedMsg:
		m.items = msg.sessions
		m.lastScan = time.Now()
		m.status = fmt.Sprintf("%d sessions", len(m.items))
		if len(msg.errors) > 0 {
			m.status = fmt.Sprintf("%s; %d provider warning(s)", m.status, len(msg.errors))
		}
		return m, m.setListItems()
	case resumeFinishedMsg:
		if msg.err != nil {
			m.status = "Resume failed: " + msg.err.Error()
		} else {
			m.status = "Resume command finished"
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.list.SettingFilter() {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(message)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			return m, m.resumeSelected()
		case "r":
			m.status = "Scanning session files…"
			return m, m.scan()
		case "a":
			m.scopeAll = !m.scopeAll
			m.status = "Scanning all projects…"
			if !m.scopeAll {
				m.status = "Scanning current directory…"
			}
			return m, m.scan()
		case "p":
			m.provider = nextProvider(m.provider)
			m.status = "Provider: " + m.provider
			return m, m.scan()
		case "s":
			if m.sortMode == config.SortActive {
				m.sortMode = config.SortPath
			} else {
				m.sortMode = config.SortActive
			}
			provider.Sort(m.items, m.sortMode)
			return m, m.setListItems()
		case "?":
			m.status = "Enter resume · / search · a scope · p provider · s sort · r refresh · q quit"
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(message)
	return m, cmd
}

func (m *Model) setListItems() tea.Cmd {
	values := make([]list.Item, 0, len(m.items))
	for _, item := range m.items {
		values = append(values, sessionItem{value: item})
	}
	return m.list.SetItems(values)
}

func (m Model) resumeSelected() tea.Cmd {
	selected, ok := m.list.SelectedItem().(sessionItem)
	if !ok {
		return func() tea.Msg { return resumeFinishedMsg{err: errors.New("no session selected")} }
	}
	spec, err := launcher.ResumeSpec(selected.value)
	if err != nil {
		return func() tea.Msg { return resumeFinishedMsg{err: err} }
	}
	command, err := launcher.Command(spec)
	if err != nil {
		return func() tea.Msg { return resumeFinishedMsg{err: err} }
	}
	return tea.ExecProcess(command, func(err error) tea.Msg { return resumeFinishedMsg{err: err} })
}

func nextProvider(current string) string {
	switch current {
	case config.ProviderAll:
		return config.ProviderClaude
	case config.ProviderClaude:
		return config.ProviderCodex
	default:
		return config.ProviderAll
	}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

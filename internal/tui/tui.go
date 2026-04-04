// Package tui provides an interactive terminal UI for browsing cheatsheets.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/bellistech/cs/internal/registry"
)

type viewState int

const (
	viewCategories viewState = iota
	viewTopics
	viewContent
)

// Model is the bubbletea model for the TUI.
type Model struct {
	reg      *registry.Registry
	state    viewState
	cursor   int
	offset   int
	width    int
	height   int
	filter   textinput.Model
	filtering bool

	// category view
	categories []string

	// topic view
	currentCat string
	topics     []*registry.Sheet
	allTopics  []*registry.Sheet // unfiltered

	// content view
	content       string
	contentLines  []string
	contentOffset int
	contentSheet  *registry.Sheet

	// status line
	status string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// New creates a new TUI model.
func New(reg *registry.Registry) Model {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 40

	return Model{
		reg:        reg,
		state:      viewCategories,
		categories: reg.Categories(),
		filter:     ti,
		status:     "j/k: move | enter: select | /: filter | q: quit | ?: help",
	}
}

// Run launches the TUI.
func Run(reg *registry.Registry) error {
	p := tea.NewProgram(New(reg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNavigation(msg)
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		m.applyFilter()
		return m, nil
	case "esc":
		m.filtering = false
		m.filter.SetValue("")
		if m.state == viewTopics {
			m.topics = m.allTopics
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilter()
		return m, cmd
	}
}

func (m *Model) applyFilter() {
	q := strings.ToLower(m.filter.Value())
	if q == "" {
		if m.state == viewTopics {
			m.topics = m.allTopics
		}
		return
	}

	switch m.state {
	case viewCategories:
		// Filter categories inline — just adjust cursor visibility
	case viewTopics:
		var filtered []*registry.Sheet
		for _, s := range m.allTopics {
			if strings.Contains(s.Name, q) || strings.Contains(strings.ToLower(s.Description), q) {
				filtered = append(filtered, s)
			}
		}
		m.topics = filtered
		m.cursor = 0
		m.offset = 0
	}
}

func (m Model) updateNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.state == viewCategories {
			return m, tea.Quit
		}
		// Go back
		return m.goBack(), nil

	case "esc":
		return m.goBack(), nil

	case "/":
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink

	case "j", "down":
		m.cursor++
		m.clampCursor()
		return m, nil

	case "k", "up":
		m.cursor--
		m.clampCursor()
		return m, nil

	case "g", "home":
		m.cursor = 0
		m.offset = 0
		if m.state == viewContent {
			m.contentOffset = 0
		}
		return m, nil

	case "G", "end":
		m.cursor = m.listLen() - 1
		if m.state == viewContent {
			max := len(m.contentLines) - m.contentHeight()
			if max < 0 {
				max = 0
			}
			m.contentOffset = max
		}
		return m, nil

	case "enter", "l", "right":
		return m.selectItem(), nil

	case "h", "left":
		return m.goBack(), nil

	case "d":
		if m.state == viewContent && m.contentSheet != nil && m.reg.HasDetail(m.contentSheet.Name) {
			d := m.reg.GetDetail(m.contentSheet.Name)
			if d != nil {
				return m.showContent(d), nil
			}
		}
		return m, nil

	case " ", "pgdown":
		if m.state == viewContent {
			m.contentOffset += m.contentHeight()
			max := len(m.contentLines) - m.contentHeight()
			if max < 0 {
				max = 0
			}
			if m.contentOffset > max {
				m.contentOffset = max
			}
		}
		return m, nil

	case "pgup":
		if m.state == viewContent {
			m.contentOffset -= m.contentHeight()
			if m.contentOffset < 0 {
				m.contentOffset = 0
			}
		}
		return m, nil

	case "?":
		m.status = "j/k: move | enter/l: select | h/esc: back | /: filter | d: detail | space: page | q: quit"
		return m, nil
	}

	return m, nil
}

func (m Model) goBack() Model {
	switch m.state {
	case viewTopics:
		m.state = viewCategories
		m.cursor = 0
		m.offset = 0
		m.filter.SetValue("")
		m.status = "j/k: move | enter: select | /: filter | q: quit"
	case viewContent:
		m.state = viewTopics
		m.contentOffset = 0
		m.status = "j/k: move | enter: select | /: filter | esc: back"
	}
	return m
}

func (m Model) selectItem() Model {
	switch m.state {
	case viewCategories:
		if m.cursor < len(m.categories) {
			cat := m.filteredCategories()[m.cursor]
			m.currentCat = cat
			m.allTopics = m.reg.ByCategory(cat)
			m.topics = m.allTopics
			m.state = viewTopics
			m.cursor = 0
			m.offset = 0
			m.filter.SetValue("")
			m.status = fmt.Sprintf("%s — j/k: move | enter: view | /: filter | esc: back", cat)
		}
	case viewTopics:
		if m.cursor < len(m.topics) {
			s := m.topics[m.cursor]
			return m.showContent(s)
		}
	}
	return m
}

func (m Model) showContent(s *registry.Sheet) Model {
	m.contentSheet = s
	m.state = viewContent
	m.contentOffset = 0

	// Render with glamour
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width-4),
	)
	if err != nil {
		m.content = s.Content
	} else {
		rendered, err := r.Render(s.Content)
		if err != nil {
			m.content = s.Content
		} else {
			m.content = rendered
		}
	}
	m.contentLines = strings.Split(m.content, "\n")

	hint := ""
	if m.reg.HasDetail(s.Name) {
		hint = " | d: deep dive"
	}
	m.status = fmt.Sprintf("%s [%s] — space/pgdn: scroll | esc: back%s", s.Name, s.Category, hint)
	return m
}

func (m *Model) clampCursor() {
	max := m.listLen() - 1
	if max < 0 {
		max = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// Adjust scroll offset
	visible := m.height - 4
	if visible < 1 {
		visible = 1
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}

func (m Model) listLen() int {
	switch m.state {
	case viewCategories:
		return len(m.filteredCategories())
	case viewTopics:
		return len(m.topics)
	case viewContent:
		return len(m.contentLines)
	}
	return 0
}

func (m Model) contentHeight() int {
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) filteredCategories() []string {
	q := strings.ToLower(m.filter.Value())
	if q == "" {
		return m.categories
	}
	var filtered []string
	for _, c := range m.categories {
		if strings.Contains(c, q) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	switch m.state {
	case viewCategories:
		sb.WriteString(titleStyle.Render("cs — Categories") + "\n\n")
		cats := m.filteredCategories()
		visible := m.height - 4
		if visible < 1 {
			visible = 1
		}
		for i := m.offset; i < len(cats) && i < m.offset+visible; i++ {
			cat := cats[i]
			count := len(m.reg.ByCategory(cat))
			line := fmt.Sprintf("  %-22s (%d sheets)", cat, count)
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString("  " + line + "\n")
			}
		}

	case viewTopics:
		sb.WriteString(titleStyle.Render(fmt.Sprintf("cs — %s", m.currentCat)) + "\n\n")
		visible := m.height - 4
		if visible < 1 {
			visible = 1
		}
		for i := m.offset; i < len(m.topics) && i < m.offset+visible; i++ {
			s := m.topics[i]
			desc := s.Description
			maxDesc := m.width - 30
			if maxDesc < 10 {
				maxDesc = 10
			}
			if len(desc) > maxDesc {
				desc = desc[:maxDesc-3] + "..."
			}
			line := fmt.Sprintf("%-22s %s", s.Name, desc)
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString("  " + line + "\n")
			}
		}

	case viewContent:
		visible := m.contentHeight()
		end := m.contentOffset + visible
		if end > len(m.contentLines) {
			end = len(m.contentLines)
		}
		for i := m.contentOffset; i < end; i++ {
			sb.WriteString(m.contentLines[i] + "\n")
		}
	}

	// Filter bar
	if m.filtering {
		sb.WriteString("\n" + m.filter.View())
	}

	// Status line
	sb.WriteString("\n" + statusStyle.Render(m.status))

	return sb.String()
}

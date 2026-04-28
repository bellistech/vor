// Package tui provides an interactive terminal UI for browsing cheatsheets.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/bellistech/vor/internal/registry"
)

type viewState int

const (
	viewCategories viewState = iota
	viewTopics
	viewContent
)

// Model is the bubbletea model for the TUI.
type Model struct {
	reg       *registry.Registry
	state     viewState
	cursor    int
	offset    int
	width     int
	height    int
	filter    textinput.Model
	filtering bool

	// filter history (persisted to ~/.cache/cs/tui-history)
	// up/down arrows recall when the filter input is focused.
	history    []string
	historyIdx int // -1 = at the live (un-recalled) input

	// help overlay (toggled with ? — independent of state)
	showHelp bool

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

// ── Amber Throne palette ────────────────────────────────────────

var (
	gold      = lipgloss.Color("#D4A017")
	purple    = lipgloss.Color("#7B2FBE")
	silver    = lipgloss.Color("#B0B0B0")
	violet    = lipgloss.Color("#C9A0DC")
	orange    = lipgloss.Color("#FF6347")
	emerald   = lipgloss.Color("#50C878")
	dimGray   = lipgloss.Color("#555555")
	darkAmber = lipgloss.Color("#8B6914")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(gold)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(gold)
	normalStyle   = lipgloss.NewStyle().Foreground(silver)
	dimStyle      = lipgloss.NewStyle().Foreground(dimGray)
	countStyle    = lipgloss.NewStyle().Foreground(violet)
	statusStyle   = lipgloss.NewStyle().Foreground(emerald)
	filterStyle   = lipgloss.NewStyle().Foreground(orange).Bold(true)
	barFull       = lipgloss.NewStyle().Foreground(gold)
	barEmpty      = lipgloss.NewStyle().Foreground(dimGray)
	borderColor   = lipgloss.NewStyle().Foreground(purple)
	posStyle      = lipgloss.NewStyle().Foreground(violet)
	descSelStyle  = lipgloss.NewStyle().Foreground(silver)
	descDimStyle  = lipgloss.NewStyle().Foreground(dimGray)
)

// ── Box-drawing helpers ─────────────────────────────────────────

func bc(s string) string { return borderColor.Render(s) }

func renderTopBorder(title string, width int) string {
	if width < 6 {
		return ""
	}
	tl := bc("╭")
	tr := bc("╮")
	sep := bc("──")

	// Build: ╭── title ──...──╮
	inner := width - 2 // minus corners
	titleRendered := sep + " " + titleStyle.Render(title) + " "
	titlePlain := "── " + title + " "
	titleLen := len(titlePlain)
	remaining := inner - titleLen
	if remaining < 0 {
		remaining = 0
	}
	fill := bc(strings.Repeat("─", remaining))
	return tl + titleRendered + fill + tr
}

func renderBottomBorder(left, right string, width int) string {
	if width < 6 {
		return ""
	}
	bl := bc("╰")
	br := bc("╯")
	sep := bc("── ")

	inner := width - 2
	leftRendered := sep + statusStyle.Render(left)
	leftPlain := "── " + left
	rightRendered := " " + posStyle.Render(right) + " " + bc("─")
	rightPlain := " " + right + " ─"

	fillLen := inner - len(leftPlain) - len(rightPlain)
	if fillLen < 0 {
		fillLen = 0
	}
	fill := bc(strings.Repeat("─", fillLen))
	return bl + leftRendered + fill + rightRendered + br
}

func renderSideBorders(lines []string, width int) string {
	left := bc("│") + " "
	right := " " + bc("│")
	innerWidth := width - 4 // 2 border + 2 padding
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Use lipgloss to pad/truncate to exact visual width
	lineStyle := lipgloss.NewStyle().Width(innerWidth)

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(left + lineStyle.Render(line) + right + "\n")
	}
	return sb.String()
}

// renderBarChart draws a proportional bar: ████░░░░
func renderBarChart(count, maxCount, width int) string {
	if maxCount == 0 || width <= 0 {
		return ""
	}
	filled := (count * width) / maxCount
	if filled == 0 && count > 0 {
		filled = 1
	}
	empty := width - filled
	return barFull.Render(strings.Repeat("█", filled)) + barEmpty.Render(strings.Repeat("░", empty))
}

// ── Bubbletea lifecycle ─────────────────────────────────────────

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
		history:    LoadHistory(),
		historyIdx: -1,
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
		// Push the committed value into history (deduped vs most-recent
		// + capped at historyMax). Save to disk best-effort; failures
		// are silently ignored — TUI usability is more important than
		// guaranteed persistence.
		m.history = pushHistory(m.history, m.filter.Value())
		m.historyIdx = -1
		_ = SaveHistory(m.history)
		m.applyFilter()
		return m, nil
	case "esc":
		m.filtering = false
		m.filter.SetValue("")
		m.historyIdx = -1
		if m.state == viewTopics {
			m.topics = m.allTopics
		}
		return m, nil
	case "up":
		// Recall older history entries. Walks from the end toward the
		// beginning. At the start of history we just stop.
		if len(m.history) == 0 {
			return m, nil
		}
		if m.historyIdx == -1 {
			m.historyIdx = len(m.history) - 1
		} else if m.historyIdx > 0 {
			m.historyIdx--
		}
		m.filter.SetValue(m.history[m.historyIdx])
		m.filter.SetCursor(len(m.history[m.historyIdx]))
		m.applyFilter()
		return m, nil
	case "down":
		// Walk back toward the live input. Past the newest entry, the
		// input clears (back to a fresh prompt).
		if len(m.history) == 0 || m.historyIdx == -1 {
			return m, nil
		}
		if m.historyIdx < len(m.history)-1 {
			m.historyIdx++
			m.filter.SetValue(m.history[m.historyIdx])
			m.filter.SetCursor(len(m.history[m.historyIdx]))
		} else {
			m.historyIdx = -1
			m.filter.SetValue("")
		}
		m.applyFilter()
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.historyIdx = -1 // typing exits history-recall mode
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
		// esc closes help overlay if open; otherwise navigates back.
		if m.showHelp {
			m.showHelp = false
			m.status = "j/k: move | enter: select | /: filter | q: quit | ?: help"
			return m, nil
		}
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
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.status = "help — press ? or esc to close"
		} else {
			m.status = "j/k: move | enter: select | /: filter | q: quit | ?: help"
		}
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

	// Render with glamour — account for border padding
	wrapWidth := m.width - 6
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
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
	// Adjust scroll offset — account for borders (top border + header gap + bottom)
	visible := m.visibleRows()
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

// visibleRows returns the number of list items that fit in the panel body.
// Panel uses: 1 top border + 1 blank + body + 1 filter (maybe) + 1 bottom border = body = height - 3
func (m Model) visibleRows() int {
	h := m.height - 3
	if m.filtering {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// contentHeight returns the number of content lines visible in the panel.
func (m Model) contentHeight() int {
	h := m.height - 3 // top border + blank line + bottom border
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

// totalSheets counts all sheets across all categories.
func (m Model) totalSheets() int {
	total := 0
	for _, c := range m.categories {
		total += len(m.reg.ByCategory(c))
	}
	return total
}

// maxCategorySize returns the largest category's sheet count (for bar scaling).
func (m Model) maxCategorySize() int {
	max := 0
	for _, c := range m.categories {
		n := len(m.reg.ByCategory(c))
		if n > max {
			max = n
		}
	}
	return max
}

// ── View ────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return m.renderHelp()
	}

	var header, footer string
	var bodyLines []string

	switch m.state {
	case viewCategories:
		header = fmt.Sprintf("cs ── The Encyclopaedia ── %d sheets", m.totalSheets())
		cats := m.filteredCategories()
		visible := m.visibleRows()
		maxCat := m.maxCategorySize()
		barWidth := 12

		for i := m.offset; i < len(cats) && i < m.offset+visible; i++ {
			cat := cats[i]
			count := len(m.reg.ByCategory(cat))
			bar := renderBarChart(count, maxCat, barWidth)
			countStr := countStyle.Render(fmt.Sprintf("(%3d)", count))

			if i == m.cursor {
				name := selectedStyle.Render(fmt.Sprintf("%-20s", cat))
				bodyLines = append(bodyLines, fmt.Sprintf(" %s %s  %s  %s",
					selectedStyle.Render("▸"), name, countStr, bar))
			} else {
				name := normalStyle.Render(fmt.Sprintf("%-20s", cat))
				bodyLines = append(bodyLines, fmt.Sprintf("   %s  %s  %s",
					name, countStr, bar))
			}
		}

		nav := "↑↓ navigate | ⏎ select | / filter | ? help"
		pos := fmt.Sprintf("%d/%d", m.cursor+1, len(cats))
		footer = renderBottomBorder(nav, pos, m.width)

	case viewTopics:
		header = fmt.Sprintf("cs ── %s ── %d sheets", m.currentCat, len(m.topics))
		visible := m.visibleRows()

		for i := m.offset; i < len(m.topics) && i < m.offset+visible; i++ {
			s := m.topics[i]
			desc := s.Description
			maxDesc := m.width - 34
			if maxDesc < 10 {
				maxDesc = 10
			}
			if len(desc) > maxDesc {
				desc = desc[:maxDesc-3] + "..."
			}

			if i == m.cursor {
				name := selectedStyle.Render(fmt.Sprintf("%-20s", s.Name))
				bodyLines = append(bodyLines, fmt.Sprintf(" %s %s  %s",
					selectedStyle.Render("▸"), name, descSelStyle.Render(desc)))
			} else {
				name := normalStyle.Render(fmt.Sprintf("%-20s", s.Name))
				bodyLines = append(bodyLines, fmt.Sprintf("   %s  %s",
					name, descDimStyle.Render(desc)))
			}
		}

		nav := "↑↓ navigate | ⏎ view | / filter | esc back"
		pos := fmt.Sprintf("%d/%d", m.cursor+1, len(m.topics))
		footer = renderBottomBorder(nav, pos, m.width)

	case viewContent:
		detailHint := ""
		if m.contentSheet != nil && m.reg.HasDetail(m.contentSheet.Name) {
			detailHint = " ── d: deep dive"
		}
		sheetName := ""
		catName := ""
		if m.contentSheet != nil {
			sheetName = m.contentSheet.Name
			catName = m.contentSheet.Category
		}
		header = fmt.Sprintf("%s ── %s%s", sheetName, catName, detailHint)

		visible := m.contentHeight()
		end := m.contentOffset + visible
		if end > len(m.contentLines) {
			end = len(m.contentLines)
		}
		for i := m.contentOffset; i < end; i++ {
			bodyLines = append(bodyLines, m.contentLines[i])
		}

		nav := "↑↓ scroll | esc back | space pgdn"
		pct := 0
		if len(m.contentLines) > visible {
			pct = (m.contentOffset * 100) / (len(m.contentLines) - visible)
		}
		pos := fmt.Sprintf("%d%%", pct)
		footer = renderBottomBorder(nav, pos, m.width)
	}

	// Build the panel
	var sb strings.Builder
	sb.WriteString(renderTopBorder(header, m.width) + "\n")
	sb.WriteString(renderSideBorders(bodyLines, m.width))

	// Filter bar (inside panel, above footer)
	if m.filtering {
		filterLine := filterStyle.Render("  filter: ") + m.filter.View()
		if m.state == viewTopics {
			filterLine += dimStyle.Render(fmt.Sprintf("  (%d matches)", len(m.topics)))
		} else if m.state == viewCategories {
			filterLine += dimStyle.Render(fmt.Sprintf("  (%d matches)", len(m.filteredCategories())))
		}
		sb.WriteString(renderSideBorders([]string{filterLine}, m.width))
	}

	sb.WriteString(footer)

	return sb.String()
}

// renderHelp returns the full-screen help overlay reachable via `?` from any
// state. Restores by pressing `?` again or `esc`.
func (m Model) renderHelp() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	header := renderTopBorder("Help — keybindings", width)
	footer := renderBottomBorder("? or esc to close", "vor TUI", width)

	help := []string{
		titleStyle.Render("Navigation"),
		"",
		"  " + selectedStyle.Render("j") + dimStyle.Render(" / ") + selectedStyle.Render("↓") +
			"      next item",
		"  " + selectedStyle.Render("k") + dimStyle.Render(" / ") + selectedStyle.Render("↑") +
			"      previous item",
		"  " + selectedStyle.Render("g") + dimStyle.Render(" / ") + selectedStyle.Render("Home") +
			"   jump to first",
		"  " + selectedStyle.Render("G") + dimStyle.Render(" / ") + selectedStyle.Render("End") +
			"    jump to last",
		"  " + selectedStyle.Render("space") + dimStyle.Render(" / ") + selectedStyle.Render("PgDn") +
			"   scroll page (in content view)",
		"  " + selectedStyle.Render("PgUp") + "        scroll back a page (in content view)",
		"",
		titleStyle.Render("Selection"),
		"",
		"  " + selectedStyle.Render("enter") + dimStyle.Render(" / ") + selectedStyle.Render("l") +
			"  open selected (category → topic → content)",
		"  " + selectedStyle.Render("h") + dimStyle.Render(" / ") + selectedStyle.Render("esc") +
			"    go back one level",
		"  " + selectedStyle.Render("d") + "          show deep-dive detail page (if available)",
		"",
		titleStyle.Render("Filter"),
		"",
		"  " + selectedStyle.Render("/") + "          start filter (live as you type)",
		"  " + selectedStyle.Render("enter") + "      commit filter",
		"  " + selectedStyle.Render("esc") + "        clear filter",
		"",
		titleStyle.Render("Other"),
		"",
		"  " + selectedStyle.Render("?") + "          toggle this help",
		"  " + selectedStyle.Render("q") + dimStyle.Render(" / ") + selectedStyle.Render("Ctrl-C") +
			" quit (from root) or back (from deeper)",
		"",
		dimStyle.Render("vör is offline-first. The default 'vor' invocation makes zero"),
		dimStyle.Render("network calls. The interactive TUI runs entirely against the"),
		dimStyle.Render("813 sheets and 722 detail pages embedded in the binary."),
		"",
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(renderSideBorders(help, width))
	sb.WriteString(footer)
	return sb.String()
}

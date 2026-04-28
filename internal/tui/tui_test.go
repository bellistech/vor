package tui

import (
	"strings"
	"testing"
	"testing/fstest"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bellistech/vor/internal/registry"
)

// testRegistry builds a small in-memory registry covering 2 categories.
// We use the same path layout the real registry expects ("sheets/<cat>/<name>.md").
func testRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	src := fstest.MapFS{
		"sheets/shell/bash.md": &fstest.MapFile{
			Data: []byte("# Bash\n\nshell.\n\n## See Also\n\n- shell/zsh\n"),
		},
		"sheets/shell/zsh.md": &fstest.MapFile{
			Data: []byte("# Zsh\n\nalt shell.\n"),
		},
		"sheets/storage/lvm.md": &fstest.MapFile{
			Data: []byte("# LVM\n\nvolume manager.\n"),
		},
		"sheets/storage/btrfs.md": &fstest.MapFile{
			Data: []byte("# Btrfs\n\nfs.\n"),
		},
	}
	reg, err := registry.New(src)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return reg
}

// keyMsg is a tiny helper to construct a tea.KeyMsg from a single rune.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// step pushes a single message and returns the resulting Model. Drops cmds.
func step(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	out, _ := m.Update(msg)
	tm, ok := out.(Model)
	if !ok {
		t.Fatalf("Update returned non-Model: %T", out)
	}
	return tm
}

func TestNew_InitialState(t *testing.T) {
	reg := testRegistry(t)
	m := New(reg)
	if m.state != viewCategories {
		t.Errorf("initial state = %d, want viewCategories(%d)", m.state, viewCategories)
	}
	if len(m.categories) != 2 {
		t.Errorf("categories = %v, want 2 (shell, storage)", m.categories)
	}
	if m.cursor != 0 || m.offset != 0 {
		t.Errorf("cursor/offset should start at 0; got %d/%d", m.cursor, m.offset)
	}
}

func TestUpdate_WindowSize(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 || m.height != 30 {
		t.Errorf("size = (%d,%d), want (100,30)", m.width, m.height)
	}
}

func TestNavigation_DownUp(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("j"))
	if m.cursor != 1 {
		t.Errorf("after j: cursor=%d, want 1", m.cursor)
	}
	m = step(t, m, keyMsg("k"))
	if m.cursor != 0 {
		t.Errorf("after k: cursor=%d, want 0", m.cursor)
	}
}

func TestNavigation_Clamped(t *testing.T) {
	m := New(testRegistry(t))
	for i := 0; i < 50; i++ {
		m = step(t, m, keyMsg("j"))
	}
	maxIdx := len(m.categories) - 1
	if m.cursor != maxIdx {
		t.Errorf("cursor should clamp to %d, got %d", maxIdx, m.cursor)
	}

	for i := 0; i < 50; i++ {
		m = step(t, m, keyMsg("k"))
	}
	if m.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", m.cursor)
	}
}

func TestNavigation_HomeEnd(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("G"))
	if m.cursor != len(m.categories)-1 {
		t.Errorf("after G: cursor=%d, want last", m.cursor)
	}
	m = step(t, m, keyMsg("g"))
	if m.cursor != 0 {
		t.Errorf("after g: cursor=%d, want 0", m.cursor)
	}
}

func TestSelect_CategoryToTopics(t *testing.T) {
	m := New(testRegistry(t))
	// shell is the first category alphabetically
	m = step(t, m, keyMsg("enter"))
	if m.state != viewTopics {
		t.Errorf("expected viewTopics, got %d", m.state)
	}
	if len(m.topics) != 2 {
		t.Errorf("shell should have 2 topics (bash, zsh); got %d", len(m.topics))
	}
	if m.currentCat != "shell" {
		t.Errorf("currentCat = %q, want shell", m.currentCat)
	}
}

func TestSelect_TopicToContent(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter"))         // pick shell
	m = step(t, m, keyMsg("enter"))         // pick first topic (bash)
	if m.state != viewContent {
		t.Errorf("expected viewContent, got %d", m.state)
	}
	if m.contentSheet == nil || m.contentSheet.Name != "bash" {
		t.Errorf("contentSheet = %+v, want bash", m.contentSheet)
	}
	if !strings.Contains(m.content, "Bash") {
		t.Errorf("content should mention Bash; got: %.100s", m.content)
	}
}

func TestGoBack_ContentToTopics(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter")) // category → topics
	m = step(t, m, keyMsg("enter")) // topic → content
	m = step(t, m, keyMsg("esc"))   // content → topics
	if m.state != viewTopics {
		t.Errorf("expected viewTopics after esc, got %d", m.state)
	}
}

func TestGoBack_TopicsToCategories(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("h"))
	if m.state != viewCategories {
		t.Errorf("expected viewCategories after h, got %d", m.state)
	}
}

func TestQuit_FromCategoriesEmitsQuitCmd(t *testing.T) {
	m := New(testRegistry(t))
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("expected non-nil cmd from q at root, got nil")
	}
	// Invoking the cmd should yield a quit-shaped message.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q at root should emit tea.QuitMsg, got %T", msg)
	}
}

func TestQuit_FromTopicsGoesBack(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter")) // → topics
	m = step(t, m, keyMsg("q"))
	if m.state != viewCategories {
		t.Errorf("q from topics should go back to categories; got %d", m.state)
	}
}

func TestFilter_EnterAndApply(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter")) // → shell topics (bash, zsh)
	m = step(t, m, keyMsg("/"))
	if !m.filtering {
		t.Fatal("expected filtering=true after /")
	}
	// type "ba" — should restrict to bash
	m = step(t, m, keyMsg("b"))
	m = step(t, m, keyMsg("a"))
	if m.filter.Value() != "ba" {
		t.Errorf("filter value = %q, want ba", m.filter.Value())
	}
	if len(m.topics) != 1 {
		t.Errorf("filter to 'ba' should leave 1 topic (bash), got %d", len(m.topics))
	}
	if m.topics[0].Name != "bash" {
		t.Errorf("filtered topic = %s, want bash", m.topics[0].Name)
	}
}

func TestFilter_EscClears(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter")) // → topics
	m = step(t, m, keyMsg("/"))
	m = step(t, m, keyMsg("z"))
	m = step(t, m, keyMsg("esc"))
	if m.filtering {
		t.Error("filtering should be false after esc")
	}
	if m.filter.Value() != "" {
		t.Errorf("filter should be cleared, got %q", m.filter.Value())
	}
	if len(m.topics) != 2 {
		t.Errorf("topics should restore to 2, got %d", len(m.topics))
	}
}

func TestFilter_EnterCommits(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("/"))
	m = step(t, m, keyMsg("z"))
	m = step(t, m, keyMsg("enter"))
	if m.filtering {
		t.Error("filtering should be false after enter (commit)")
	}
	if len(m.topics) != 1 || m.topics[0].Name != "zsh" {
		t.Errorf("after committed filter z: expected [zsh], got %v", m.topics)
	}
}

func TestContent_PageDown(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("enter"))
	if m.state != viewContent {
		t.Fatalf("expected content state, got %d", m.state)
	}
	// give the model a small content size so contentHeight() is meaningful
	m.height = 20
	m = step(t, m, keyMsg("space"))
	if m.contentOffset < 0 {
		t.Errorf("content offset should not go negative: %d", m.contentOffset)
	}
}

func TestView_DoesNotPanic(t *testing.T) {
	m := New(testRegistry(t))
	m.width = 100
	m.height = 30

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() panicked at categories state: %v", r)
		}
	}()
	_ = m.View()

	m = step(t, m, keyMsg("enter")) // → topics
	_ = m.View()

	m = step(t, m, keyMsg("enter")) // → content
	_ = m.View()
}

func TestHelpKeyTogglesOverlay(t *testing.T) {
	m := New(testRegistry(t))
	if m.showHelp {
		t.Fatal("expected showHelp=false initially")
	}
	m = step(t, m, keyMsg("?"))
	if !m.showHelp {
		t.Error("? should turn help overlay on")
	}
	if !strings.Contains(m.status, "help") {
		t.Errorf("status should mention help, got: %q", m.status)
	}
	// Press ? again — should toggle off.
	m = step(t, m, keyMsg("?"))
	if m.showHelp {
		t.Error("? again should turn help overlay off")
	}
}

func TestHelpEscClosesOverlay(t *testing.T) {
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("?"))
	if !m.showHelp {
		t.Fatal("expected help on after ?")
	}
	prevState := m.state
	m = step(t, m, keyMsg("esc"))
	if m.showHelp {
		t.Error("esc should close help overlay")
	}
	if m.state != prevState {
		t.Errorf("esc with help open should NOT navigate back; state=%d, want %d", m.state, prevState)
	}
}

func TestHelpRendersFullScreen(t *testing.T) {
	m := New(testRegistry(t))
	m.width = 100
	m.height = 30
	m = step(t, m, keyMsg("?"))

	view := m.View()
	for _, want := range []string{
		"Help",
		"keybindings",
		"Navigation",
		"Filter",
		"j",
		"enter",
		"toggle this help",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("help view missing %q\n--- view ---\n%s", want, view)
		}
	}
}

func TestInit_ReturnsNoCmd(t *testing.T) {
	m := New(testRegistry(t))
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() should return nil cmd, got %v", cmd)
	}
}

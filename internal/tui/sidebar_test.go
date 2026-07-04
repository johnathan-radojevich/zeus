package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func drillToServerActions(m Model) Model {
	m = loadServersInto(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	return update(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func TestCollapseSidebarsKeepsNavigationStack(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = drillToServerActions(m)
	if m.PanelCount() != 4 {
		t.Fatalf("expected 4 panels in stack, got %d", m.PanelCount())
	}

	if !m.SidebarsCollapsed() {
		t.Fatal("expected collapsed sidebars by default")
	}
	if m.PanelCount() != 4 {
		t.Fatalf("expected stack unchanged after collapse, got %d panels", m.PanelCount())
	}
	if m.visibleSidebarCount() != 1 {
		t.Fatalf("expected 1 visible sidebar, got %d", m.visibleSidebarCount())
	}
}

func TestCollapsedViewShowsPathAndOneSidebar(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = drillToServerActions(m)

	view := m.View()
	if !strings.Contains(view, "home › server utilities › prod-us-east-1 › actions") &&
		!strings.Contains(view, "actions") {
		t.Fatal("expected full path in header")
	}
	if m.CurrentSelection() != "deploy" {
		t.Fatalf("expected focused panel selection deploy, got %q", m.CurrentSelection())
	}
}

func TestToggleSidebarsCommand(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeySpace})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.SidebarsCollapsed() {
		t.Fatal("expected expand sidebars command to expand from default collapsed state")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeySpace})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.SidebarsCollapsed() {
		t.Fatal("expected collapse sidebars command to collapse again")
	}
}

func TestCollapsedSidebarClickUsesFocusedPanel(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = drillToServerActions(m)

	topBars := m.topBarLines()
	itemY := topBars + headerHeight + 1
	m, _ = apply(m, tea.MouseMsg{
		X:      5,
		Y:      itemY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.CurrentSelection() == "server utilities" {
		t.Fatal("expected click in collapsed sidebar to target focused panel, not home")
	}
}

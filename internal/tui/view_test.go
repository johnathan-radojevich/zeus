package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathBarInView(t *testing.T) {
	m := New()
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected multiline view, got %q", view)
	}
	if !strings.Contains(lines[0], "nautlius") {
		t.Fatalf("expected path bar on first line, first line=%q", lines[0])
	}
	if lipgloss.Height(view) > m.height {
		t.Fatalf("view height %d exceeds terminal height %d", lipgloss.Height(view), m.height)
	}
}

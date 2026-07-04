package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAdaptiveThemeDark(t *testing.T) {
	lipgloss.SetHasDarkBackground(true)
	t.Cleanup(func() { lipgloss.SetHasDarkBackground(false) })

	if !IsDarkMode() {
		t.Fatal("expected dark mode")
	}
	view := themedView()
	if !strings.Contains(view, "nautlius") {
		t.Fatalf("expected dark view to render, got %q", view[:min(80, len(view))])
	}
}

func TestAdaptiveThemeLight(t *testing.T) {
	lipgloss.SetHasDarkBackground(false)

	if IsDarkMode() {
		t.Fatal("expected light mode")
	}
	view := themedView()
	if !strings.Contains(view, "nautlius") {
		t.Fatalf("expected light view to render, got %q", view[:min(80, len(view))])
	}
}

func TestAdaptivePalettePairs(t *testing.T) {
	pairs := []struct {
		name  string
		light string
		dark  string
	}{
		{"accent", colorAccent.Light, colorAccent.Dark},
		{"text", colorText.Light, colorText.Dark},
		{"border", colorBorder.Light, colorBorder.Dark},
	}
	for _, p := range pairs {
		if p.light == p.dark {
			t.Fatalf("%s: light and dark palette values should differ", p.name)
		}
	}
}

func themedView() string {
	m := New()
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	return m.View()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

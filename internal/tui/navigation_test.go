package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestVimNavigateList(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	first := m.CurrentSelection()
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.CurrentSelection() == first {
		t.Fatal("expected j to move selection down")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.CurrentSelection() != first {
		t.Fatalf("expected k to restore %q, got %q", first, m.CurrentSelection())
	}
}

func TestVimOpenAndBack(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.PanelCount() != 2 {
		t.Fatalf("expected 2 panels after l, got %d", m.PanelCount())
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.PanelCount() != 1 {
		t.Fatalf("expected 1 panel after h, got %d", m.PanelCount())
	}
}

func TestVimKeysDoNotStartFilter(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	for _, key := range []rune{'j', 'k', 'h', 'l'} {
		m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		if m.panels[0].list.FilterState() != list.Unfiltered {
			t.Fatalf("expected %q not to start filter", string(key))
		}
	}
}

func TestGoToEndWithVim(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.CurrentSelection() != "codebase navigation" {
		t.Fatalf("expected G to jump to last item, got %q", m.CurrentSelection())
	}
}

package tui

import (
	"strings"
	"testing"

	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCodebaseMenuOptions(t *testing.T) {
	menu := CodebaseMenu()
	if len(menu) != 2 {
		t.Fatalf("expected 2 codebase options, got %d", len(menu))
	}
	if menu[0].Title != findXMLForRuleKeyTitle {
		t.Fatalf("unexpected first option: %q", menu[0].Title)
	}
	if menu[0].Action != ActionFindXMLForRuleKey {
		t.Fatalf("unexpected first action: %q", menu[0].Action)
	}
	if menu[1].Title != findImplementingClassTitle {
		t.Fatalf("unexpected second option: %q", menu[1].Title)
	}
	if menu[1].Action != ActionFindImplementingClass {
		t.Fatalf("unexpected second action: %q", menu[1].Action)
	}
}

func openCodebaseNavigation(m Model) Model {
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	return update(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func openFindImplementingClass(m Model) Model {
	m = openCodebaseNavigation(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	return update(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func TestRuleKeySearchOpensEditor(t *testing.T) {
	setHelixAvailableForTest(func() bool { return false })
	defer resetHelixEditor()

	setFindXMLForRuleKeyForTest(func(key string) (services.RuleXMLFile, bool) {
		return services.RuleXMLFile{
			Path:    "rules/access-control.xml",
			Content: "<rule key=\"" + key + "\">\n  <name>sample</name>\n</rule>\n",
		}, true
	})
	defer resetFindXMLForRuleKey()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = m.openRuleKeySearch()
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("MY-RULE-001")})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.codebaseMode != codebaseXMLEditor {
		t.Fatalf("expected xml editor mode, got %d", m.codebaseMode)
	}
	if m.xmlEditorPath != "rules/access-control.xml" {
		t.Fatalf("unexpected path: %q", m.xmlEditorPath)
	}
}

func TestRuleKeySearchNotFound(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = m.openRuleKeySearch()
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing-key")})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.codebaseActive() {
		t.Fatal("expected codebase mode to close after failed search")
	}
	if !strings.Contains(m.Status(), "no xml found") {
		t.Fatalf("expected not-found status, got %q", m.Status())
	}
}

func TestClassSearchShowsResults(t *testing.T) {
	setFindImplementingClassForTest(func(query string) ([]services.ImplementingClass, bool) {
		return []services.ImplementingClass{
			{Name: "DefaultPolicyValidator", Path: "internal/policy/validator.go"},
			{Name: "StrictPolicyValidator", Path: "internal/policy/strict.go"},
		}, true
	})
	defer resetFindImplementingClass()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = m.openClassSearch()
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("PolicyValidator")})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.codebaseMode != codebaseClassResults {
		t.Fatalf("expected class results mode, got %d", m.codebaseMode)
	}
	if len(m.classSearchResults) != 2 {
		t.Fatalf("expected 2 results, got %d", len(m.classSearchResults))
	}

	view := m.View()
	if !strings.Contains(view, "DefaultPolicyValidator") {
		t.Fatal("expected first result in view")
	}
}

func TestClassSearchNotFound(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = openFindImplementingClass(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("MissingType")})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.codebaseActive() {
		t.Fatal("expected codebase mode to close after failed search")
	}
	if !strings.Contains(m.Status(), "no implementing class found") {
		t.Fatalf("expected not-found status, got %q", m.Status())
	}
}

func TestClassSearchOpensFromMenu(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = openFindImplementingClass(m)
	if m.codebaseMode != codebaseClassInput {
		t.Fatalf("expected class input mode, got %d", m.codebaseMode)
	}
}

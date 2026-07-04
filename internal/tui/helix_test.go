package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWriteTempXML(t *testing.T) {
	path, err := writeTempXML(scratchXMLTemplate)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<rule>") {
		t.Fatalf("unexpected temp content: %q", content)
	}
}

func TestScratchXMLEditorFallbackWhenHelixMissing(t *testing.T) {
	setHelixAvailableForTest(func() bool { return false })
	defer resetHelixEditor()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = openCodebaseNavigation(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.codebaseMode != codebaseXMLEditor {
		t.Fatalf("expected built-in editor fallback, got mode %d", m.codebaseMode)
	}
	if !strings.Contains(m.Status(), "hx not found") {
		t.Fatalf("expected helix missing status, got %q", m.Status())
	}
}

func TestScratchXMLEditorLaunchesHelix(t *testing.T) {
	setHelixAvailableForTest(func() bool { return true })

	var launchedPath string
	setRunHelixEditorForTest(func(path string) tea.Cmd {
		launchedPath = path
		return func() tea.Msg {
			return helixFinishedMsg{path: path}
		}
	})
	defer resetHelixEditor()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = openCodebaseNavigation(m)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected helix launch command")
	}

	msg := cmd()
	finished, ok := msg.(helixFinishedMsg)
	if !ok {
		t.Fatalf("expected helixFinishedMsg, got %T", msg)
	}
	if launchedPath == "" {
		t.Fatal("expected helix to receive a temp path")
	}
	if finished.path != launchedPath {
		t.Fatalf("path mismatch: %q vs %q", finished.path, launchedPath)
	}

	m = update(m, finished)
	if !strings.Contains(m.Status(), "edited xml") {
		t.Fatalf("expected edited status, got %q", m.Status())
	}
	if _, err := os.Stat(launchedPath); !os.IsNotExist(err) {
		t.Fatal("expected temp file to be removed after helix exits")
	}
}

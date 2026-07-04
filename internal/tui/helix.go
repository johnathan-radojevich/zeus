package tui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

type helixFinishedMsg struct {
	path string
	err  error
}

var (
	helixBinary    = "hx"
	helixAvailable = defaultHelixAvailable
	runHelixEditor = defaultRunHelixEditor
)

func defaultHelixAvailable() bool {
	_, err := exec.LookPath(helixBinary)
	return err == nil
}

func defaultRunHelixEditor(path string) tea.Cmd {
	c := exec.Command(helixBinary, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return helixFinishedMsg{path: path, err: err}
	})
}

func setHelixAvailableForTest(fn func() bool) {
	helixAvailable = fn
}

func setRunHelixEditorForTest(fn func(string) tea.Cmd) {
	runHelixEditor = fn
}

func resetHelixEditor() {
	helixBinary = "hx"
	helixAvailable = defaultHelixAvailable
	runHelixEditor = defaultRunHelixEditor
}

func writeTempXML(content string) (string, error) {
	f, err := os.CreateTemp("", "nautlius-*.xml")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func scratchXMLFile() services.RuleXMLFile {
	return services.RuleXMLFile{
		Path:    scratchXMLPath,
		Content: scratchXMLTemplate,
	}
}

func (m Model) launchXMLEdit(file services.RuleXMLFile) (Model, tea.Cmd) {
	path, err := writeTempXML(file.Content)
	if err != nil {
		m.status = "failed to create temp file: " + err.Error()
		return m, nil
	}

	if !helixAvailable() {
		next := m.openXMLEditor(file)
		next.status = "hx not found — using built-in editor"
		os.Remove(path)
		return next, nil
	}

	m.status = "editing in helix · " + path
	return m, runHelixEditor(path)
}

func (m Model) handleHelixFinished(msg helixFinishedMsg) (Model, tea.Cmd) {
	defer os.Remove(msg.path)

	if msg.err != nil {
		m.status = "helix exited with error"
		return m, nil
	}

	content, err := os.ReadFile(msg.path)
	if err != nil {
		m.status = "failed to read edited file: " + err.Error()
		return m, nil
	}

	m.status = fmt.Sprintf("edited xml (%d bytes)", len(content))
	return m, nil
}

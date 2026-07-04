package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	findXMLForRuleKeyTitle       = "find xml for rule key"
	findImplementingClassTitle   = "find implementing class"
	ActionFindXMLForRuleKey      = "find-xml-for-rule-key"
	ActionFindImplementingClass  = "find-implementing-class"
	scratchXMLPath               = "scratch.xml"
)

const scratchXMLTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rule>

</rule>
`

type codebaseMode int

const (
	codebaseInactive codebaseMode = iota
	codebaseRuleKeyInput
	codebaseClassInput
	codebaseClassResults
	codebaseXMLEditor
)

var (
	findXMLForRuleKey       = services.FindXMLForRuleKey
	findImplementingClass   = services.FindImplementingClass
)

func setFindXMLForRuleKeyForTest(fn func(string) (services.RuleXMLFile, bool)) {
	findXMLForRuleKey = fn
}

func resetFindXMLForRuleKey() {
	findXMLForRuleKey = services.FindXMLForRuleKey
}

func setFindImplementingClassForTest(fn func(string) ([]services.ImplementingClass, bool)) {
	findImplementingClass = fn
}

func resetFindImplementingClass() {
	findImplementingClass = services.FindImplementingClass
}

func (m Model) codebaseActive() bool {
	return m.codebaseMode != codebaseInactive
}

func (m Model) openScratchXMLEditor() (Model, tea.Cmd) {
	return m.launchXMLEdit(scratchXMLFile())
}

func (m Model) openRuleKeySearch() Model {
	m.codebaseMode = codebaseRuleKeyInput
	m.ruleKeyInput = newRuleKeyInput()
	m.ruleKeyInput.Focus()
	m.status = "enter a rule key to search"
	return m
}

func (m Model) openClassSearch() Model {
	m.codebaseMode = codebaseClassInput
	m.classSearchInput = newClassSearchInput()
	m.classSearchResults = nil
	m.classSearchInput.Focus()
	m.status = "enter a class or interface to search"
	return m
}

func (m Model) closeCodebaseMode() Model {
	m.codebaseMode = codebaseInactive
	m.ruleKeyInput.Blur()
	m.classSearchInput.Blur()
	m.classSearchResults = nil
	m.xmlEditor.Blur()
	m.xmlEditorPath = ""
	return m
}

func (m Model) openXMLEditor(file services.RuleXMLFile) Model {
	width, height := m.mainPaneEditorSize()
	m.codebaseMode = codebaseXMLEditor
	m.xmlEditorPath = file.Path
	m.xmlEditor = newXMLEditor(file.Content, width, height)
	m.xmlEditor.Focus()
	m.status = "scratch xml · esc to close"
	return m
}

func newRuleKeyInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "rule key"
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = "> "
	return ti
}

func newClassSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "class or interface"
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = "> "
	return ti
}

func newXMLEditor(content string, width, height int) textarea.Model {
	ta := textarea.New()
	ta.SetValue(content)
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}
	ta.SetWidth(width)
	ta.SetHeight(height)
	return ta
}

func (m Model) mainPaneEditorSize() (int, int) {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	mainWidth := innerWidth - m.visibleSidebarWidth()
	if mainWidth < 20 {
		mainWidth = 20
	}
	width := mainWidth - 6
	height := m.panelHeight() - 6
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}
	return width, height
}

func (m Model) resizeCodebaseEditor() Model {
	if m.codebaseMode != codebaseXMLEditor {
		return m
	}
	width, height := m.mainPaneEditorSize()
	m.xmlEditor.SetWidth(width)
	m.xmlEditor.SetHeight(height)
	return m
}

func (m Model) handleCodebaseMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch m.codebaseMode {
	case codebaseRuleKeyInput:
		return m.handleRuleKeyInputMsg(msg)
	case codebaseClassInput:
		return m.handleClassSearchInputMsg(msg)
	case codebaseClassResults:
		return m.handleClassResultsMsg(msg)
	case codebaseXMLEditor:
		return m.handleXMLEditorMsg(msg)
	}
	return m, nil
}

func (m Model) handleRuleKeyInputMsg(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m = m.closeCodebaseMode()
		m.status = ""
		return m, nil
	case "enter":
		return m.submitRuleKeySearch()
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.ruleKeyInput, cmd = m.ruleKeyInput.Update(keyMsg)
	return m, cmd
}

func (m Model) submitRuleKeySearch() (Model, tea.Cmd) {
	query := strings.TrimSpace(m.ruleKeyInput.Value())
	file, ok := findXMLForRuleKey(query)
	if !ok {
		m.status = "no xml found for rule key " + query
		return m.closeCodebaseMode(), nil
	}
	next, cmd := m.launchXMLEdit(file)
	next.ruleKeyInput.Blur()
	if cmd != nil {
		next.codebaseMode = codebaseInactive
	}
	return next, cmd
}

func (m Model) handleClassSearchInputMsg(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m = m.closeCodebaseMode()
		m.status = ""
		return m, nil
	case "enter":
		return m.submitClassSearch()
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.classSearchInput, cmd = m.classSearchInput.Update(keyMsg)
	return m, cmd
}

func (m Model) submitClassSearch() (Model, tea.Cmd) {
	query := strings.TrimSpace(m.classSearchInput.Value())
	results, ok := findImplementingClass(query)
	if !ok {
		m.status = "no implementing class found for " + query
		return m.closeCodebaseMode(), nil
	}

	m.classSearchInput.Blur()
	m.classSearchResults = results
	m.codebaseMode = codebaseClassResults
	m.status = fmt.Sprintf("found %d implementations", len(results))
	return m, nil
}

func (m Model) handleClassResultsMsg(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m = m.closeCodebaseMode()
		m.status = ""
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleXMLEditorMsg(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m = m.closeCodebaseMode()
			return m, nil
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.xmlEditor, cmd = m.xmlEditor.Update(msg)
	return m, cmd
}

func renderRuleKeySearchPane(width, height int, input textinput.Model) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}

	header := labelStyle.Render("find xml for rule key")
	hint := hintStyle.Render("enter rule key · esc to cancel")
	body := header + "\n" + hint + "\n\n" + input.View()
	return mainPaneStyle.Width(width).Height(height).Render(body)
}

func renderClassSearchPane(width, height int, input textinput.Model) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}

	header := labelStyle.Render(findImplementingClassTitle)
	hint := hintStyle.Render("enter class or interface · esc to cancel")
	body := header + "\n" + hint + "\n\n" + input.View()
	return mainPaneStyle.Width(width).Height(height).Render(body)
}

func renderClassResultsPane(width, height int, results []services.ImplementingClass) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}

	lines := []string{
		labelStyle.Render(findImplementingClassTitle),
		hintStyle.Render("esc to close"),
		"",
	}
	for _, r := range results {
		lines = append(lines, statusStyle.Render(r.Name))
		lines = append(lines, hintStyle.Render(r.Path))
		lines = append(lines, "")
	}
	body := strings.Join(lines, "\n")
	return mainPaneStyle.Width(width).Height(height).Render(body)
}

func renderXMLEditorPane(width, height int, path string, editor textarea.Model) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}

	header := labelStyle.Render(strings.ToLower(path))
	hint := hintStyle.Render("esc to close")
	body := header + "\n" + hint + "\n\n" + editor.View()
	return mainPaneStyle.Width(width).Height(height).Render(body)
}

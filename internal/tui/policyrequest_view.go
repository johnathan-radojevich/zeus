package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type policyRequestEditorFocus int

const (
	policyRequestFocusNone policyRequestEditorFocus = iota
	policyRequestFocusHeaders
	policyRequestFocusBody
	policyRequestFocusResponse
)

type policyRequestLayout struct {
	sectionWidth   int
	editorWidth    int
	headersHeight  int
	bodyHeight     int
	responseHeight int
}

func (m Model) policyRequestEditing() bool {
	return m.onRunRequestPanel() && (m.policyRequestResult != nil || m.policyRequestSelectedEndpoint != "")
}

func (m Model) mainPaneSize() (int, int) {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	mainWidth := innerWidth - m.visibleSidebarWidth()
	if mainWidth < 20 {
		mainWidth = 20
	}
	return mainWidth, m.panelHeight()
}

const (
	policyRequestMetaLines        = 4
	policyRequestSectionChrome    = 3
	policyRequestResponseExtra    = 1 // search row above viewport
	policyRequestHeadersMinLines  = 5
	policyRequestProgressLines    = 3 // status + bar + spacer
)

func (m Model) policyRequestTopExtraLines() int {
	if m.policyRequestRunning || m.policyRequestBodyLoading {
		return policyRequestProgressLines
	}
	return 0
}

func layoutContentLines(l policyRequestLayout, extraTopLines int) int {
	return extraTopLines + policyRequestMetaLines +
		l.headersHeight + policyRequestSectionChrome +
		l.bodyHeight + policyRequestSectionChrome +
		l.responseHeight + policyRequestSectionChrome + policyRequestResponseExtra
}

func computePolicyRequestLayout(outerW, outerH, extraTopLines int) policyRequestLayout {
	if outerW < 30 {
		outerW = 30
	}
	if outerH < 16 {
		outerH = 16
	}

	innerW := outerW - 6
	if innerW < 20 {
		innerW = 20
	}

	maxContent := outerH - 4
	if maxContent < 12 {
		maxContent = 12
	}

	headersHeight := policyRequestHeadersMinLines
	fixed := extraTopLines + policyRequestMetaLines + headersHeight + policyRequestSectionChrome
	remaining := maxContent - fixed
	if remaining < 8 {
		remaining = 8
	}

	half := remaining / 2
	bodyHeight := half - policyRequestSectionChrome
	responseHeight := half - policyRequestSectionChrome - policyRequestResponseExtra

	if bodyHeight < 3 {
		bodyHeight = 3
	}
	if responseHeight < 4 {
		responseHeight = 4
	}

	return policyRequestLayout{
		sectionWidth:   innerW,
		editorWidth:    max(16, innerW-4),
		headersHeight:  headersHeight,
		bodyHeight:     bodyHeight,
		responseHeight: responseHeight,
	}
}

func newPolicyRequestTextarea(content string, width, height int) textarea.Model {
	ta := textarea.New()
	ta.SetValue(content)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetWidth(width)
	ta.SetHeight(height)
	return ta
}

func newPolicyResponseSearch(width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "search response"
	ti.CharLimit = 128
	if width > 8 {
		ti.Width = width - 8
	} else {
		ti.Width = 20
	}
	ti.Prompt = "/ "
	return ti
}

func newPolicyResponseViewport(content string, width, height int) viewport.Model {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}
	vp := viewport.New(width, height)
	vp.MouseWheelEnabled = true
	vp.SetContent(content)
	return vp
}

func findMatchLines(content, query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	lower := strings.ToLower(content)
	q := strings.ToLower(query)

	var lines []int
	start := 0
	for {
		idx := strings.Index(lower[start:], q)
		if idx < 0 {
			break
		}
		idx += start
		lines = append(lines, strings.Count(content[:idx], "\n"))
		start = idx + len(q)
	}
	return lines
}

func highlightResponseMatches(content, query string, activeLine int) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return content
	}

	lower := strings.ToLower(content)
	q := strings.ToLower(query)
	var b strings.Builder
	start := 0

	for {
		idx := strings.Index(lower[start:], q)
		if idx < 0 {
			b.WriteString(content[start:])
			break
		}
		idx += start
		b.WriteString(content[start:idx])

		match := content[idx : idx+len(query)]
		line := strings.Count(content[:idx], "\n")
		style := listFilterMatchStyle
		if line == activeLine {
			style = listSelectedTitleStyle
		}
		b.WriteString(style.Render(match))
		start = idx + len(query)
	}
	return b.String()
}

func (m Model) openPolicyRequestDraft(headers, body string) Model {
	width, height := m.mainPaneSize()
	layout := computePolicyRequestLayout(width, height, m.policyRequestTopExtraLines())

	m.policyRequestHeadersEditor = newPolicyRequestTextarea(headers, layout.editorWidth, layout.headersHeight)
	m.policyRequestBodyEditor = newPolicyRequestTextarea(body, layout.editorWidth, layout.bodyHeight)

	response := "press r to run"
	m.policyRequestResponseRaw = response
	m.policyRequestResponseSearch = newPolicyResponseSearch(layout.editorWidth)
	m.policyRequestResponseSearchActive = false
	m.policyRequestResponseMatchIndex = 0
	m.policyRequestResponseViewport = newPolicyResponseViewport(response, layout.editorWidth, layout.responseHeight)

	m.policyRequestEditorFocus = policyRequestFocusNone
	m.policyRequestHeadersEditor.Blur()
	m.policyRequestBodyEditor.Blur()
	m.policyRequestResponseSearch.Blur()
	return m
}

func (m Model) openPolicyRequestEditors(result services.PolicyRequestResult) Model {
	width, height := m.mainPaneSize()
	layout := computePolicyRequestLayout(width, height, m.policyRequestTopExtraLines())

	headers := services.FormatHTTPHeaders(result.RequestHeaders)
	m.policyRequestHeadersEditor = newPolicyRequestTextarea(headers, layout.editorWidth, layout.headersHeight)
	m.policyRequestBodyEditor = newPolicyRequestTextarea(result.RequestBody, layout.editorWidth, layout.bodyHeight)

	response := result.ResponseBody
	if response == "" {
		response = "empty"
	}
	m.policyRequestResponseRaw = response
	m.policyRequestResponseSearch = newPolicyResponseSearch(layout.editorWidth)
	m.policyRequestResponseSearchActive = false
	m.policyRequestResponseMatchIndex = 0
	m.policyRequestResponseViewport = newPolicyResponseViewport(response, layout.editorWidth, layout.responseHeight)

	m.policyRequestEditorFocus = policyRequestFocusNone
	m.policyRequestHeadersEditor.Blur()
	m.policyRequestBodyEditor.Blur()
	m.policyRequestResponseSearch.Blur()
	return m
}

func (m Model) syncResponseViewport() Model {
	query := m.policyRequestResponseSearch.Value()
	lines := findMatchLines(m.policyRequestResponseRaw, query)
	activeLine := 0
	if len(lines) > 0 {
		if m.policyRequestResponseMatchIndex >= len(lines) {
			m.policyRequestResponseMatchIndex = 0
		}
		activeLine = lines[m.policyRequestResponseMatchIndex]
	}
	display := highlightResponseMatches(m.policyRequestResponseRaw, query, activeLine)
	m.policyRequestResponseViewport.SetContent(display)
	if len(lines) > 0 {
		m.policyRequestResponseViewport.YOffset = max(0, activeLine)
	}
	return m
}

func (m Model) scrollResponseToMatch(index int) Model {
	lines := findMatchLines(m.policyRequestResponseRaw, m.policyRequestResponseSearch.Value())
	if len(lines) == 0 {
		m.policyRequestResponseMatchIndex = 0
		return m.syncResponseViewport()
	}
	if index < 0 {
		index = len(lines) - 1
	}
	if index >= len(lines) {
		index = 0
	}
	m.policyRequestResponseMatchIndex = index
	m = m.syncResponseViewport()
	return m
}

func (m Model) nextResponseSearchMatch() Model {
	return m.scrollResponseToMatch(m.policyRequestResponseMatchIndex + 1)
}

func (m Model) prevResponseSearchMatch() Model {
	return m.scrollResponseToMatch(m.policyRequestResponseMatchIndex - 1)
}

func (m Model) activateResponseSearch() Model {
	m.policyRequestResponseSearchActive = true
	m.policyRequestResponseSearch.Focus()
	m.policyRequestResponseMatchIndex = 0
	return m.syncResponseViewport()
}

func (m Model) deactivateResponseSearch() Model {
	m.policyRequestResponseSearch.SetValue("")
	m.policyRequestResponseSearchActive = false
	m.policyRequestResponseSearch.Blur()
	m.policyRequestResponseMatchIndex = 0
	return m.syncResponseViewport()
}

func (m Model) blurPolicyRequestEditors() Model {
	m.policyRequestEditorFocus = policyRequestFocusNone
	m.policyRequestHeadersEditor.Blur()
	m.policyRequestBodyEditor.Blur()
	m = m.deactivateResponseSearch()
	return m
}

func (m Model) focusPolicyRequestEditor(focus policyRequestEditorFocus) Model {
	m.policyRequestEditorFocus = focus
	m.policyRequestHeadersEditor.Blur()
	m.policyRequestBodyEditor.Blur()
	m.policyRequestResponseSearch.Blur()
	m.policyRequestResponseSearchActive = false

	switch focus {
	case policyRequestFocusHeaders:
		m.policyRequestHeadersEditor.Focus()
	case policyRequestFocusBody:
		m.policyRequestBodyEditor.Focus()
	case policyRequestFocusResponse:
		m.policyRequestResponseSearchActive = false
	}
	return m
}

func (m Model) cyclePolicyRequestEditorFocus(delta int) Model {
	order := []policyRequestEditorFocus{
		policyRequestFocusHeaders,
		policyRequestFocusBody,
		policyRequestFocusResponse,
	}
	current := 0
	for i, f := range order {
		if f == m.policyRequestEditorFocus {
			current = i
			break
		}
	}
	if m.policyRequestEditorFocus == policyRequestFocusNone {
		if delta >= 0 {
			return m.focusPolicyRequestEditor(order[0])
		}
		return m.focusPolicyRequestEditor(order[len(order)-1])
	}
	next := (current + delta) % len(order)
	if next < 0 {
		next += len(order)
	}
	return m.focusPolicyRequestEditor(order[next])
}

func (m Model) applyPolicyRequestLayout(layout policyRequestLayout) Model {
	m.policyRequestHeadersEditor.SetWidth(layout.editorWidth)
	m.policyRequestHeadersEditor.SetHeight(layout.headersHeight)
	m.policyRequestBodyEditor.SetWidth(layout.editorWidth)
	m.policyRequestBodyEditor.SetHeight(layout.bodyHeight)
	m.policyRequestResponseViewport.Width = layout.editorWidth
	m.policyRequestResponseViewport.Height = layout.responseHeight
	m.policyRequestResponseSearch.Width = max(12, layout.editorWidth-8)
	return m
}

func (m Model) resizePolicyRequestEditors() Model {
	if m.policyRequestResult == nil && m.policyRequestSelectedEndpoint == "" {
		return m
	}
	width, height := m.mainPaneSize()
	layout := computePolicyRequestLayout(width, height, m.policyRequestTopExtraLines())
	return m.applyPolicyRequestLayout(layout)
}

func (m Model) policyRequestSectionFocused() bool {
	return m.policyRequestEditorFocus != policyRequestFocusNone
}

func (m Model) policyRequestEditorsFocused() bool {
	return m.policyRequestSectionFocused()
}

func (m Model) handleResponsePaneKey(keyMsg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.policyRequestResponseSearchActive {
		switch keyMsg.String() {
		case "esc":
			return m.deactivateResponseSearch(), nil, true
		case "ctrl+n":
			return m.nextResponseSearchMatch(), nil, true
		case "ctrl+p":
			return m.prevResponseSearchMatch(), nil, true
		case "enter":
			return m.nextResponseSearchMatch(), nil, true
		}
		old := m.policyRequestResponseSearch.Value()
		var cmd tea.Cmd
		m.policyRequestResponseSearch, cmd = m.policyRequestResponseSearch.Update(keyMsg)
		if m.policyRequestResponseSearch.Value() != old {
			m = m.scrollResponseToMatch(0)
		}
		return m, cmd, true
	}

	switch keyMsg.String() {
	case "/":
		return m.activateResponseSearch(), textinput.Blink, true
	}

	var cmd tea.Cmd
	m.policyRequestResponseViewport, cmd = m.policyRequestResponseViewport.Update(keyMsg)
	return m, cmd, true
}

// handlePolicyRequestPaneMsg handles keys when a result is visible.
// Returns handled=true when the message was consumed.
func (m Model) handlePolicyRequestPaneMsg(msg tea.Msg) (Model, tea.Cmd, bool) {
	keyMsg, isKey := msg.(tea.KeyMsg)

	if isKey {
		switch keyMsg.String() {
		case "n", "e":
			if m.policyRequestResponseSearchActive {
				break
			}
			if m.policyRequestEditorFocus == policyRequestFocusHeaders ||
				m.policyRequestEditorFocus == policyRequestFocusBody {
				break
			}
			policy, ok := m.activePolicyFromStack()
			if ok {
				txn := services.PolicyTransactionNewBusiness
				if keyMsg.String() == "e" {
					txn = services.PolicyTransactionEndorsement
				}
				return m.applyTransactionType(policy.PolicyNumber, txn), nil, true
			}
		case "1":
			return m.focusPolicyRequestEditor(policyRequestFocusHeaders), nil, true
		case "2":
			return m.focusPolicyRequestEditor(policyRequestFocusBody), nil, true
		case "3":
			return m.focusPolicyRequestEditor(policyRequestFocusResponse), nil, true
		case "tab":
			return m.cyclePolicyRequestEditorFocus(1), nil, true
		case "shift+tab":
			return m.cyclePolicyRequestEditorFocus(-1), nil, true
		case "esc":
			if m.policyRequestResponseSearchActive {
				return m.deactivateResponseSearch(), nil, true
			}
			if m.policyRequestSectionFocused() {
				return m.blurPolicyRequestEditors(), nil, true
			}
			return m, nil, false
		case "r":
			if m.policyRequestResult == nil && m.policyRequestSelectedEndpoint != "" &&
				!m.policyRequestSectionFocused() && !m.policyRequestBodyLoading {
				next, cmd := m.runSelectedPolicyRequest()
				return next, cmd, true
			}
		case "b":
			if m.policyRequestResult == nil && m.policyRequestSelectedEndpoint != "" &&
				!m.policyRequestSectionFocused() && !m.policyRequestBodyLoading {
				next, cmd := m.refreshPolicyRequestBodyFromSource()
				return next, cmd, true
			}
		case "x":
			if !m.policyRequestSectionFocused() {
				return m.exportPolicyRequestBruno(), nil, true
			}
		case "s":
			if !m.policyRequestSectionFocused() {
				next, cmd := m.openRequestServerPicker()
				return next, cmd, true
			}
		case "ctrl+c", "q":
			return m, tea.Quit, true
		}
	}

	if m.policyRequestEditorFocus == policyRequestFocusResponse {
		if isKey {
			return m.handleResponsePaneKey(keyMsg)
		}
		if m.policyRequestResponseSearchActive {
			old := m.policyRequestResponseSearch.Value()
			var cmd tea.Cmd
			m.policyRequestResponseSearch, cmd = m.policyRequestResponseSearch.Update(msg)
			if m.policyRequestResponseSearch.Value() != old {
				m = m.scrollResponseToMatch(0)
			}
			return m, cmd, true
		}
		var cmd tea.Cmd
		m.policyRequestResponseViewport, cmd = m.policyRequestResponseViewport.Update(msg)
		return m, cmd, true
	}

	if !m.policyRequestSectionFocused() ||
		(m.policyRequestEditorFocus != policyRequestFocusHeaders && m.policyRequestEditorFocus != policyRequestFocusBody) {
		return m, nil, false
	}

	if !isKey {
		return m, nil, false
	}

	var cmd tea.Cmd
	switch m.policyRequestEditorFocus {
	case policyRequestFocusHeaders:
		m.policyRequestHeadersEditor, cmd = m.policyRequestHeadersEditor.Update(msg)
	case policyRequestFocusBody:
		m.policyRequestBodyEditor, cmd = m.policyRequestBodyEditor.Update(msg)
	}
	return m, cmd, true
}

func renderStatusBadge(code int) string {
	switch {
	case code >= 400:
		return statusBadgeWarnStyle.Render(fmt.Sprintf(" %d ", code))
	default:
		return statusBadgeOKStyle.Render(fmt.Sprintf(" %d ", code))
	}
}

func renderRequestSectionLabel(title string, focused bool) string {
	if focused {
		return labelStyle.Copy().Foreground(colorAccent).Render("▸ " + title)
	}
	return labelStyle.Render(title)
}

func renderRequestEditorSection(title string, focused bool, editor textarea.Model, width int, hint string) string {
	label := renderRequestSectionLabel(title, focused)
	if hint != "" {
		label += hintStyle.Render(" · " + hint)
	}
	style := requestSectionStyle
	if focused {
		style = requestSectionActiveStyle
	}
	return style.Width(width).Render(label + "\n" + editor.View())
}

func renderResponseSection(m Model, focused bool, layout policyRequestLayout) string {
	label := renderRequestSectionLabel("response", focused)
	style := requestSectionStyle
	if focused {
		style = requestSectionActiveStyle
	}

	var searchRow string
	switch {
	case m.policyRequestResponseSearchActive:
		count := len(findMatchLines(m.policyRequestResponseRaw, m.policyRequestResponseSearch.Value()))
		suffix := ""
		if count > 0 {
			suffix = hintStyle.Render(fmt.Sprintf(" · %d/%d", m.policyRequestResponseMatchIndex+1, count))
		}
		searchRow = m.policyRequestResponseSearch.View() + suffix
	case focused:
		searchRow = hintStyle.Render("/ search · j/k scroll · ctrl+n/p next match")
	default:
		searchRow = hintStyle.Render("3 focus · scroll · / search")
	}

	body := label + "\n" + searchRow + "\n" + m.policyRequestResponseViewport.View()
	return style.Width(layout.sectionWidth).Render(body)
}

func (m Model) renderPolicyRequestPane(width, height int) string {
	layout := computePolicyRequestLayout(width, height, m.policyRequestTopExtraLines())

	headersEditor := m.policyRequestHeadersEditor
	headersEditor.SetWidth(layout.editorWidth)
	headersEditor.SetHeight(layout.headersHeight)
	bodyEditor := m.policyRequestBodyEditor
	bodyEditor.SetWidth(layout.editorWidth)
	bodyEditor.SetHeight(layout.bodyHeight)

	vp := m.policyRequestResponseViewport
	vp.Width = layout.editorWidth
	vp.Height = layout.responseHeight

	var endpointName, targetURL, serverName string
	var statusBadge string
	hasResult := m.policyRequestResult != nil
	if hasResult {
		result := *m.policyRequestResult
		endpointName = result.EndpointName
		targetURL = result.TargetURL
		serverName = result.ServerName
		statusBadge = renderStatusBadge(result.StatusCode)
	} else {
		endpointName = m.policyRequestDraftEndpointName
		targetURL = m.policyRequestDraftTargetURL
		if policy, ok := m.activePolicyFromStack(); ok {
			serverName = m.policyRequestServer(policy.PolicyNumber)
		}
		statusBadge = hintStyle.Render(" draft ")
	}

	focus := m.policyRequestEditorFocus
	txnLabel := services.PolicyTransactionNewBusiness.Label()
	if policy, ok := m.activePolicyFromStack(); ok {
		txnLabel = m.policyTransactionType(policy.PolicyNumber).Label()
	}

	meta := lipgloss.JoinHorizontal(lipgloss.Top,
		emptyIconStyle.Render("◈")+" ",
		requestMetaStyle.Render(endpointName),
		statusBadge,
	)
	methodLine := requestMethodStyle.Render("POST") + hintStyle.Render(" "+targetURL)
	if !m.policyRequestRunning && !m.policyRequestBodyLoading {
		methodLine += hintStyle.Render(" · x export")
	}
	if !hasResult && !m.policyRequestRunning && !m.policyRequestBodyLoading {
		methodLine += hintStyle.Render(" · r run")
	}
	serverLine := hintStyle.Render("server · ") + statusStyle.Render(serverName) + hintStyle.Render(" · s change")
	if serverName == "" {
		serverLine = hintStyle.Render("server · select a target · s change")
	}
	txnLine := hintStyle.Render("transaction · ") + statusStyle.Render(txnLabel) + hintStyle.Render(" · n/e change")

	headersHint := "1 focus · tab cycle"
	bodyHint := "2 focus"
	if !hasResult && !m.policyRequestBodyLoading {
		bodyHint += " · b refresh"
	}

	headersSection := renderRequestEditorSection(
		"headers",
		focus == policyRequestFocusHeaders,
		headersEditor,
		layout.sectionWidth,
		headersHint,
	)
	bodySection := renderRequestEditorSection(
		"body",
		focus == policyRequestFocusBody,
		bodyEditor,
		layout.sectionWidth,
		bodyHint,
	)
	responseSection := renderResponseSection(m, focus == policyRequestFocusResponse, layout)

	sections := []string{
		meta,
		methodLine,
		serverLine,
		txnLine,
		headersSection,
		bodySection,
		responseSection,
	}
	if m.policyRequestRunning || m.policyRequestBodyLoading {
		m.policyRequestProgress.Width = m.mainPaneBarWidth()
		progress := labelStyle.Render(strings.ToLower(m.status)) + "\n" + m.policyRequestProgress.View()
		sections = append([]string{progress, ""}, sections...)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return mainPaneStyle.Width(width).Height(height).Render(content)
}

func (m Model) renderPolicyRunRequestSetupPane(width, height int) string {
	if width < 10 {
		width = 10
	}
	if height < 8 {
		height = 8
	}

	policy, ok := m.activePolicyFromStack()
	txn := services.PolicyTransactionNewBusiness
	policyLine := hintStyle.Render("select a policy to configure requests")
	if ok {
		txn = m.policyTransactionType(policy.PolicyNumber)
		policyLine = requestMetaStyle.Render(policy.Title) +
			hintStyle.Render(" · "+policy.PolicyNumber+" · "+policy.ControlNumber)
	}

	sections := []string{
		labelStyle.Render("run request"),
		policyLine,
		"",
		labelStyle.Render("transaction type"),
		statusStyle.Render(txn.Label()) + hintStyle.Render(" · n/e change"),
		"",
		labelStyle.Render("target server"),
		m.runRequestSetupServerLabelStyled() + hintStyle.Render(" · s change"),
		"",
		labelStyle.Render("endpoints"),
		hintStyle.Render("select in sidebar · enter to preview · r to run"),
	}
	content := strings.Join(sections, "\n")
	return mainPaneStyle.Width(width).Height(height).Render(content)
}

func (m Model) runRequestSetupServerLabelStyled() string {
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return statusStyle.Render("none selected")
	}
	if name := m.policyRequestServer(policy.PolicyNumber); name != "" {
		return statusStyle.Render(name)
	}
	return statusStyle.Render("loading…")
}

func (m Model) handlePolicyRequestClick(x, y int) (Model, tea.Cmd) {
	width, height := m.mainPaneSize()
	layout := computePolicyRequestLayout(width, height, m.policyRequestTopExtraLines())

	metaLines := policyRequestMetaLines + m.policyRequestTopExtraLines()
	headersEnd := metaLines + layout.headersHeight + policyRequestSectionChrome
	bodyEnd := headersEnd + layout.bodyHeight + policyRequestSectionChrome
	responseEnd := bodyEnd + layout.responseHeight + policyRequestSectionChrome + policyRequestResponseExtra

	switch {
	case y >= metaLines && y < headersEnd:
		return m.focusPolicyRequestEditor(policyRequestFocusHeaders), nil
	case y >= headersEnd && y < bodyEnd:
		return m.focusPolicyRequestEditor(policyRequestFocusBody), nil
	case y >= bodyEnd && y < responseEnd:
		return m.focusPolicyRequestEditor(policyRequestFocusResponse), nil
	default:
		_ = x
		_ = height
		return m.blurPolicyRequestEditors(), nil
	}
}

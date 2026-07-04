package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	panels             []sidebarPanel
	sidebarsCollapsed  bool
	width              int
	height             int
	status             string
	ready              bool
	commandMode      commandMode
	commandPalette   commandPalette
	policySearch     textinput.Model
	servers          []MenuItem
	serversLoading   bool
	serversFetchedAt time.Time
	serverProgress   progress.Model
	codebaseMode        codebaseMode
	ruleKeyInput        textinput.Model
	classSearchInput    textinput.Model
	classSearchResults  []services.ImplementingClass
	xmlEditor           textarea.Model
	xmlEditorPath       string
	policyEndpointsPolicy  string
	policyEndpoints        []MenuItem
	policyEndpointsLoading   bool
	policyRequestRunning     bool
	policyRequestBodyLoading bool
	policyRequestProgress    progress.Model
	policyRequestResult         *services.PolicyRequestResult
	policyRequestErr            string
	policyRequestHeadersEditor       textarea.Model
	policyRequestBodyEditor          textarea.Model
	policyRequestResponseViewport    viewport.Model
	policyRequestResponseSearch      textinput.Model
	policyRequestResponseRaw         string
	policyRequestResponseSearchActive bool
	policyRequestResponseMatchIndex  int
	policyRequestSelectedEndpoint    string
	policyRequestDraftEndpointName   string
	policyRequestDraftTargetURL      string
	policyRequestEditorFocus         policyRequestEditorFocus
	policyTransactionTypes      map[string]services.PolicyTransactionType
	policyRequestServers        map[string]string
	serverPickerOpen            bool
	serverPicker                serverPicker
}

func New() Model {
	panel := newSidebarPanel("home", RootMenu(nil, false), 24)

	return Model{
		panels:            []sidebarPanel{panel},
		sidebarsCollapsed: true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m = m.resizePanels()
		m = m.resizeCodebaseEditor()
		m = m.resizePolicyRequestEditors()
		if m.serverPickerOpen {
			m = m.refreshServerPicker()
		}
		return m, nil

	case helixFinishedMsg:
		return m.handleHelixFinished(msg)

	case policyEndpointsLoadedMsg:
		return m.handlePolicyEndpointsLoaded(msg)

	case policyRequestResultMsg:
		return m.handlePolicyRequestResult(msg)

	case policyRequestBodyLoadedMsg:
		return m.handlePolicyRequestBodyLoaded(msg)

	case serversLoadedMsg:
		return m.handleServersLoaded(msg)

	case serverProgressTickMsg:
		return m.handleServerProgressTick()

	case progress.FrameMsg:
		if m.serversLoading {
			var cmd tea.Cmd
			updated, cmd := m.serverProgress.Update(msg)
			m.serverProgress = updated.(progress.Model)
			return m, cmd
		}
		if m.policyEndpointsLoading || m.policyRequestRunning || m.policyRequestBodyLoading {
			var cmd tea.Cmd
			updated, cmd := m.policyRequestProgress.Update(msg)
			m.policyRequestProgress = updated.(progress.Model)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.serverPickerOpen {
			next, cmd := m.handleServerPickerKey(msg)
			return next, cmd
		}

		if m.commandActive() {
			next, cmd := m.handleCommandKey(msg)
			return next, cmd
		}

		if m.codebaseActive() {
			return m.handleCodebaseMsg(msg)
		}

		if m.policyRequestEditing() {
			next, cmd, handled := m.handlePolicyRequestPaneMsg(msg)
			if handled {
				return next, cmd
			}
		}

		focus := len(m.panels) - 1
		panel := m.panels[focus]

		if m.onRunRequestPanel() && !m.policyRequestRunning && !panel.filtering() {
			if policy, ok := m.activePolicyFromStack(); ok {
				switch msg.String() {
				case "n":
					return m.applyTransactionType(policy.PolicyNumber, services.PolicyTransactionNewBusiness), nil
				case "e":
					return m.applyTransactionType(policy.PolicyNumber, services.PolicyTransactionEndorsement), nil
				}
			}
			if msg.String() == "s" {
				return m.openRequestServerPicker()
			}
		}

		if msg.String() == "r" && m.onServerUtilitiesPanel() && !panel.filtering() {
			return m.refreshServers()
		}

		if msg.String() == "r" && m.onRunRequestPanel() && !panel.filtering() {
			return m.runSelectedPolicyRequest()
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ":
			return m.openCommandMenu(), nil
		case "enter", "l", "right":
			if panel.filtering() {
				var cmd tea.Cmd
				panel.list, cmd = panel.list.Update(msg)
				m.panels[focus] = panel
				return m, cmd
			}
			return m.selectCurrent()
		case "esc", "backspace":
			if panel.filtering() {
				var cmd tea.Cmd
				panel.list, cmd = panel.list.Update(msg)
				m.panels[focus] = panel
				return m, cmd
			}
			return m.popPanel()
		case "h", "left":
			return m.popPanel()
		}

		if isTypingFilterKey(msg) {
			panel, cmd := panel.beginTypingFilter(msg)
			m.panels[focus] = panel
			return m, cmd
		}

	case tea.MouseMsg:
		if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
			return m, nil
		}
		return m.handleClick(msg.X, msg.Y)
	}

	if m.policyRequestSectionFocused() {
		var cmd tea.Cmd
		switch m.policyRequestEditorFocus {
		case policyRequestFocusHeaders:
			m.policyRequestHeadersEditor, cmd = m.policyRequestHeadersEditor.Update(msg)
		case policyRequestFocusBody:
			m.policyRequestBodyEditor, cmd = m.policyRequestBodyEditor.Update(msg)
		case policyRequestFocusResponse:
			if m.policyRequestResponseSearchActive {
				old := m.policyRequestResponseSearch.Value()
				m.policyRequestResponseSearch, cmd = m.policyRequestResponseSearch.Update(msg)
				if m.policyRequestResponseSearch.Value() != old {
					m = m.scrollResponseToMatch(0)
				}
			} else {
				m.policyRequestResponseViewport, cmd = m.policyRequestResponseViewport.Update(msg)
			}
		}
		return m, cmd
	}

	focus := len(m.panels) - 1
	panel := m.panels[focus]
	var cmd tea.Cmd
	panel.list, cmd = panel.list.Update(msg)
	m.panels[focus] = panel
	return m, cmd
}

func (m Model) selectedPolicy() (MenuItem, bool) {
	for i := len(m.panels) - 1; i >= 0; i-- {
		item, ok := m.panels[i].selectedItem()
		if ok && item.IsPolicy() {
			return item, true
		}
	}
	return MenuItem{}, false
}

func (m Model) topBarLines() int {
	lines := pathBarLines(m.width, m.panels)
	if policy, ok := m.selectedPolicy(); ok {
		lines += policyDetailBarLines(m.width, policy)
	}
	return lines
}

func (m Model) View() string {
	if !m.ready {
		return loadingStyle.Render(" loading… ")
	}

	pathBar := renderPathBar(m.width, m.panels)
	var header []string
	header = append(header, pathBar)
	if policy, ok := m.selectedPolicy(); ok {
		header = append(header, renderPolicyDetailBar(m.width, policy))
	}
	headerRow := lipgloss.JoinVertical(lipgloss.Left, header...)

	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	footer := renderFooter(innerWidth, m.showServerRefreshHint())

	if m.commandActive() {
		return m.renderCommandView(headerRow, footer)
	}

	content := m.renderMainLayout(headerRow, footer)
	if m.serverPickerOpen {
		return renderModalOverlay(content, m.renderServerPickerDialog(), m.width)
	}
	return content
}

func (m Model) renderMainLayout(headerRow, footer string) string {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	panelHeight := m.panelHeight()

	var sidebars []string
	visible := m.visibleSidebarPanels()
	for i, panel := range visible {
		style := inactiveSidebarStyle
		if i == len(visible)-1 {
			style = activeSidebarStyle
		}
		rendered := style.Width(sidebarWidth).Height(panelHeight).Render(panel.view())
		sidebars = append(sidebars, rendered)
	}

	sidebarRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebars...)
	mainWidth := innerWidth - m.visibleSidebarWidth()
	if mainWidth < 20 {
		mainWidth = 20
	}

	main := m.renderMainPane(mainWidth, panelHeight)
	row := lipgloss.JoinHorizontal(lipgloss.Top, sidebarRow, main)

	return lipgloss.JoinVertical(lipgloss.Left, headerRow, appStyle.Render(row), footer)
}

func (m Model) contentHeight() int {
	barLines := m.topBarLines()
	footLines := footerLines(max(1, m.width-2))
	h := m.height - barLines - footLines
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) panelHeight() int {
	h := m.contentHeight() - 2
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) resizePanels() Model {
	panelHeight := m.panelHeight()
	listHeight := panelHeight - 2
	if listHeight < 1 {
		listHeight = 1
	}

	for i, panel := range m.panels {
		panel.list.SetHeight(listHeight)
		panel.list.SetWidth(sidebarWidth - 4)
		m.panels[i] = panel
	}
	return m
}

func (m Model) selectCurrent() (Model, tea.Cmd) {
	focus := len(m.panels) - 1
	item, ok := m.panels[focus].selectedItem()
	if !ok {
		return m, nil
	}
	return m.activate(focus, item)
}

func (m Model) activate(panelIdx int, item MenuItem) (Model, tea.Cmd) {
	m.panels = m.panels[:panelIdx+1]

	if item.HasChildren() {
		if item.Title == serverUtilitiesTitle {
			m, cmd := m.ensureServersLoaded()
			children := serverUtilitiesChildren(m.servers, m.serversLoading)
			next := newSidebarPanel(item.Title, children, m.contentHeight())
			m.panels = append(m.panels, next)
			return m, cmd
		}

		if item.Title == runRequestTitle {
			policy, ok := m.activePolicyFromStack()
			if !ok {
				m.status = "no policy selected"
				return m, nil
			}
			m = m.ensureDefaultRequestServer(policy.PolicyNumber)
			m, serverCmd := m.ensureServersLoaded()
			m, endpointCmd := m.ensurePolicyEndpoints(policy)
			children := runRequestPanelChildren(m.policyEndpoints, m.policyEndpointsLoading)
			next := newSidebarPanel(runRequestTitle, children, m.contentHeight())
			m.panels = append(m.panels, next)
			return m, tea.Batch(serverCmd, endpointCmd)
		}

		next := newSidebarPanel(item.Title, item.Children, m.contentHeight())
		m.panels = append(m.panels, next)
		m = m.syncPolicyRequestDisplay()
		return m, nil
	}

	if item.IsEndpoint() {
		policy, ok := m.activePolicyFromStack()
		if !ok {
			m.status = "no policy selected"
			return m, nil
		}
		m = m.preparePolicyRequest(policy, item)
		return m, nil
	}

	if item.Action != "" {
		if item.Action == ActionFindXMLForRuleKey {
			return m.openScratchXMLEditor()
		}
		if item.Action == ActionFindImplementingClass {
			m = m.openClassSearch()
			return m, nil
		}
		m.status = item.Action
	}
	m = m.syncPolicyRequestDisplay()
	return m, nil
}

func (m Model) popPanel() (Model, tea.Cmd) {
	if len(m.panels) <= 1 {
		return m, nil
	}
	if m.onRunRequestPanel() {
		m = m.clearPolicyRequestDisplay()
	}
	m.panels = m.panels[:len(m.panels)-1]
	m = m.syncPolicyRequestDisplay()
	return m, nil
}

func (m Model) visibleSidebarPanels() []sidebarPanel {
	if !m.sidebarsCollapsed || len(m.panels) == 0 {
		return m.panels
	}
	return m.panels[len(m.panels)-1:]
}

func (m Model) visibleSidebarCount() int {
	if m.sidebarsCollapsed {
		return 1
	}
	return len(m.panels)
}

func (m Model) visibleSidebarWidth() int {
	return m.visibleSidebarCount() * sidebarWidth
}

func (m Model) toggleSidebarsCollapsed() Model {
	m.sidebarsCollapsed = !m.sidebarsCollapsed
	if m.sidebarsCollapsed {
		m.status = "sidebars collapsed"
	} else {
		m.status = "sidebars expanded"
	}
	return m.resizePanels().resizeCodebaseEditor()
}

func (m Model) handleClick(x, y int) (Model, tea.Cmd) {
	topBars := m.topBarLines()
	if y < topBars {
		return m, nil
	}

	sidebarPixels := m.visibleSidebarCount()*sidebarWidth + 1
	if m.policyRequestEditing() && x >= sidebarPixels {
		return m.handlePolicyRequestClick(x-sidebarPixels, y-topBars-1)
	}

	panelIdx := panelIndexAt(x, m.sidebarsCollapsed, len(m.panels))
	if panelIdx < 0 || panelIdx >= len(m.panels) {
		return m, nil
	}

	itemIdx := itemIndexAt(y, topBars)
	visible := m.panels[panelIdx].list.VisibleItems()
	if itemIdx < 0 || itemIdx >= len(visible) {
		return m, nil
	}

	m.panels = m.panels[:panelIdx+1]
	m.panels[panelIdx] = m.panels[panelIdx].setIndex(itemIdx)

	item, ok := m.panels[panelIdx].selectedItem()
	if !ok {
		return m, nil
	}
	return m.activate(panelIdx, item)
}

// Run starts the TUI program.
func Run() error {
	initTheme()
	p := tea.NewProgram(New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// SelectedPath returns the titles of the currently open panels (useful for tests).
func (m Model) SelectedPath() []string {
	path := make([]string, len(m.panels))
	for i, p := range m.panels {
		path[i] = p.title
	}
	return path
}

// Status returns the last leaf action message.
func (m Model) Status() string {
	return m.status
}

// PanelCount returns how many panels are in the navigation stack.
func (m Model) PanelCount() int {
	return len(m.panels)
}

// SidebarsCollapsed reports whether only the focused sidebar is rendered.
func (m Model) SidebarsCollapsed() bool {
	return m.sidebarsCollapsed
}

// CurrentSelection returns the focused panel's selected item title.
func (m Model) CurrentSelection() string {
	if len(m.panels) == 0 {
		return ""
	}
	item, ok := m.panels[len(m.panels)-1].selectedItem()
	if !ok {
		return ""
	}
	return item.Title
}

// Breadcrumb returns the navigation path as a single string.
func (m Model) Breadcrumb() string {
	return strings.Join(m.SelectedPath(), " › ")
}

func (m Model) showServerRefreshHint() bool {
	return m.onServerUtilitiesPanel()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

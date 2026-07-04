package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type commandMode int

const (
	commandClosed commandMode = iota
	commandMenu
	commandPolicySearch
)

type commandID string

const (
	cmdSearchPolicy     commandID = "search-policy"
	cmdRefreshServers   commandID = "refresh-servers"
	cmdToggleSidebars   commandID = "toggle-sidebars"
)

type commandEntry struct {
	id          commandID
	title       string
	description string
}

type commandItem struct {
	entry commandEntry
}

func (c commandItem) Title() string       { return c.entry.title }
func (c commandItem) Description() string { return c.entry.description }
func (c commandItem) FilterValue() string { return c.entry.title }

type commandPalette struct {
	list  list.Model
	items []commandEntry
}

func commandsFor(m Model) []commandEntry {
	collapseTitle := "collapse sidebars"
	collapseDesc := "show one sidebar; keep path in header"
	if m.sidebarsCollapsed {
		collapseTitle = "expand sidebars"
		collapseDesc = "show nested sidebars side by side"
	}
	return []commandEntry{
		{
			id:          cmdSearchPolicy,
			title:       "search policy",
			description: "find by policy or control number",
		},
		{
			id:          cmdRefreshServers,
			title:       "refresh servers",
			description: "reload server list from source",
		},
		{
			id:          cmdToggleSidebars,
			title:       collapseTitle,
			description: collapseDesc,
		},
	}
}

func newCommandPalette(width, height int, m Model) commandPalette {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = listNormalTitleStyle
	delegate.Styles.DimmedTitle = listDimmedTitleStyle
	delegate.Styles.SelectedTitle = listSelectedTitleStyle

	entries := commandsFor(m)
	listItems := make([]list.Item, len(entries))
	for i, entry := range entries {
		listItems[i] = commandItem{entry: entry}
	}

	if width < 20 {
		width = 20
	}
	if height < 4 {
		height = 4
	}

	l := list.New(listItems, delegate, width-4, height-4)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return commandPalette{list: l, items: entries}
}

func (c commandPalette) selected() (commandEntry, bool) {
	selected := c.list.SelectedItem()
	if selected == nil {
		return commandEntry{}, false
	}
	item, ok := selected.(commandItem)
	if !ok {
		return commandEntry{}, false
	}
	return item.entry, true
}

func (c commandPalette) view(title string) string {
	header := labelStyle.Render(title)
	return header + "\n" + c.list.View()
}

func (c *commandPalette) setSize(width, height int) {
	if width < 20 {
		width = 20
	}
	if height < 4 {
		height = 4
	}
	c.list.SetWidth(width - 4)
	c.list.SetHeight(height - 4)
}

func newPolicySearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "policy or control number"
	ti.CharLimit = 32
	ti.Width = 30
	ti.Prompt = "> "
	return ti
}

func renderCommandPanel(width, height int, palette commandPalette) string {
	if width < 1 {
		width = 1
	}
	palette.setSize(width, height)
	content := palette.view("commands")
	return commandMenuStyle.Width(width).Height(height).Render(content)
}

func renderPolicySearchPanel(width, height int, input textinput.Model) string {
	if width < 1 {
		width = 1
	}
	header := labelStyle.Render("search policy")
	hint := hintStyle.Render("enter policy number (e.g. POL-1042) or control number (e.g. CTRL-AC-042)")
	body := header + "\n" + hint + "\n\n" + input.View()
	return commandMenuStyle.Width(width).Height(height).Render(body)
}

func commandPanelHeight(mode commandMode, contentHeight int) int {
	h := contentHeight - 2
	if h < 6 {
		h = 6
	}
	if mode == commandPolicySearch {
		if h < 8 {
			h = 8
		}
	}
	return h
}

func (m Model) openCommandMenu() Model {
	width := m.width - 2
	if width < 1 {
		width = 1
	}
	height := commandPanelHeight(commandMenu, m.contentHeight())
	m.commandPalette = newCommandPalette(width, height, m)
	m.commandMode = commandMenu
	return m
}

func (m Model) openPolicySearch() Model {
	m.commandMode = commandPolicySearch
	m.policySearch = newPolicySearchInput()
	m.policySearch.Focus()
	return m
}

func (m Model) closeCommand() Model {
	m.commandMode = commandClosed
	m.policySearch.Blur()
	return m
}

func (m Model) handleCommandKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.commandMode == commandPolicySearch {
			return m.openCommandMenu(), nil
		}
		return m.closeCommand(), nil
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.commandMode {
	case commandMenu:
		return m.handleCommandMenuKey(msg)
	case commandPolicySearch:
		return m.handlePolicySearchKey(msg)
	}
	return m, nil
}

func (m Model) handleCommandMenuKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "l", "right":
		entry, ok := m.commandPalette.selected()
		if !ok {
			return m, nil
		}
		switch entry.id {
		case cmdSearchPolicy:
			return m.openPolicySearch(), textinput.Blink
		case cmdRefreshServers:
			next, cmd := m.refreshServers()
			next = next.closeCommand()
			return next, cmd
		case cmdToggleSidebars:
			next := m.toggleSidebarsCollapsed().closeCommand()
			return next, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.commandPalette.list, cmd = m.commandPalette.list.Update(msg)
	return m, cmd
}

func (m Model) handlePolicySearchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.policySearch.Value())
		policy, ok := FindPolicy(query)
		if !ok {
			m.status = "no policy found for " + query
			return m.closeCommand(), nil
		}
		return m.navigateToPolicy(policy), nil
	}

	var cmd tea.Cmd
	m.policySearch, cmd = m.policySearch.Update(msg)
	return m, cmd
}

func (m Model) navigateToPolicy(item MenuItem) Model {
	h := m.panelHeight()
	policies := allPolicies()
	policyIdx := 0
	for i, p := range policies {
		if p.PolicyNumber == item.PolicyNumber {
			policyIdx = i
			break
		}
	}

	m.panels = []sidebarPanel{
		newSidebarPanel("home", m.rootMenu(), h),
		newSidebarPanel("policy tools", PolicyToolsMenu(), h),
		newSidebarPanel("policies", policiesSubmenu(), h),
		newSidebarPanel("view all", policies, h).setIndex(policyIdx),
	}
	m.status = "found policy " + item.Title
	m.commandMode = commandClosed
	return m.resizePanels()
}

func policiesSubmenu() []MenuItem {
	return PolicyToolsMenu()[0].Children
}

func (m Model) commandActive() bool {
	return m.commandMode != commandClosed
}

func (m Model) renderCommandView(headerRow, footer string) string {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	panelHeight := commandPanelHeight(m.commandMode, m.contentHeight())

	var body string
	switch m.commandMode {
	case commandMenu:
		body = renderCommandPanel(innerWidth, panelHeight, m.commandPalette)
	case commandPolicySearch:
		body = renderPolicySearchPanel(innerWidth, panelHeight, m.policySearch)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerRow, appStyle.Render(body), footer)
}

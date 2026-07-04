package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type serverPickerEntry struct {
	name        string
	environment string
	region      string
}

type serverPickerColumns struct {
	name   int
	env    int
	region int
}

type serverPicker struct {
	search    textinput.Model
	entries   []serverPickerEntry
	selected  int
	scrollTop int
}

func menuItemToServerEntry(item MenuItem) serverPickerEntry {
	env, region := parseServerDescription(item.Description)
	return serverPickerEntry{
		name:        item.Title,
		environment: env,
		region:      region,
	}
}

func parseServerDescription(desc string) (environment, region string) {
	parts := strings.Split(desc, " · ")
	if len(parts) > 0 {
		environment = parts[0]
	}
	if len(parts) > 1 {
		region = parts[1]
	}
	return environment, region
}

func newServerPickerSearch(tableWidth int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "filter servers"
	ti.Prompt = "> "
	ti.CharLimit = 64
	if tableWidth > 12 {
		ti.Width = tableWidth
	} else {
		ti.Width = 20
	}
	return ti
}

func serverPickerColumnLayout() serverPickerColumns {
	return serverPickerColumns{
		name:   24,
		env:    16,
		region: 14,
	}
}

func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := runewidth.StringWidth(s)
	if w >= width {
		return truncateVisual(s, 0, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func (c serverPickerColumns) totalWidth() int {
	return c.name + c.env + c.region + 4
}

func renderServerPickerHeader(cols serverPickerColumns) string {
	name := labelStyle.Render(padRight("server", cols.name))
	env := labelStyle.Render(padRight("environment", cols.env))
	region := labelStyle.Render(padRight("region", cols.region))
	return lipgloss.JoinHorizontal(lipgloss.Top, name, "  ", env, "  ", region)
}

func renderServerPickerRow(entry serverPickerEntry, cols serverPickerColumns, selected bool) string {
	name := padRight(entry.name, cols.name)
	env := padRight(entry.environment, cols.env)
	region := padRight(entry.region, cols.region)
	row := lipgloss.JoinHorizontal(lipgloss.Top, name, "  ", env, "  ", region)
	if selected {
		return listSelectedTitleStyle.Width(cols.totalWidth()).Render(row)
	}
	return listNormalTitleStyle.Render(row)
}

func (m Model) serverPickerDialogSize() (dialogWidth, listHeight int) {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	contentH := m.contentHeight()
	if contentH < 1 {
		contentH = 1
	}

	dialogWidth = innerWidth * 4 / 5
	if dialogWidth > innerWidth-6 {
		dialogWidth = innerWidth - 6
	}
	if dialogWidth < 72 {
		dialogWidth = min(72, innerWidth-4)
	}

	chromeLines := 11
	dialogHeight := contentH * 3 / 4
	if dialogHeight < 28 {
		dialogHeight = min(28, contentH-2)
	}
	if dialogHeight > contentH-2 {
		dialogHeight = contentH - 2
	}

	listHeight = dialogHeight - chromeLines
	if listHeight < 16 {
		listHeight = min(16, contentH-chromeLines)
	}
	if listHeight < 4 {
		listHeight = 4
	}
	return dialogWidth, listHeight
}

func serverPickerEntriesFromMenu(servers []MenuItem, loading bool) []serverPickerEntry {
	switch {
	case loading && len(servers) == 0:
		return nil
	case len(servers) == 0:
		return nil
	default:
		entries := make([]serverPickerEntry, len(servers))
		for i, s := range servers {
			entries[i] = menuItemToServerEntry(s)
		}
		return entries
	}
}

func filterServerPickerEntries(entries []serverPickerEntry, query string) []serverPickerEntry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entries
	}
	filtered := make([]serverPickerEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.name), query) ||
			strings.Contains(strings.ToLower(entry.environment), query) ||
			strings.Contains(strings.ToLower(entry.region), query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func newServerPicker(dialogWidth, listHeight int, servers []MenuItem, loading bool, searchText, selectedName string) serverPicker {
	cols := serverPickerColumnLayout()

	picker := serverPicker{
		search:  newServerPickerSearch(cols.totalWidth()),
		entries: serverPickerEntriesFromMenu(servers, loading),
	}
	picker.search.SetValue(searchText)
	picker.search.Focus()
	picker.selectByName(selectedName, listHeight)
	return picker
}

func (p *serverPicker) filtered() []serverPickerEntry {
	return filterServerPickerEntries(p.entries, p.search.Value())
}

func (p *serverPicker) selectByName(name string, visibleRows int) {
	if name == "" {
		p.selected = 0
		p.scrollTop = 0
		return
	}
	filtered := p.filtered()
	for i, entry := range filtered {
		if entry.name == name {
			p.selected = i
			p.ensureScroll(visibleRows, len(filtered))
			return
		}
	}
	p.selected = 0
	p.scrollTop = 0
}

func (p *serverPicker) selectedName() (string, bool) {
	filtered := p.filtered()
	if p.selected < 0 || p.selected >= len(filtered) {
		return "", false
	}
	return filtered[p.selected].name, true
}

func (p *serverPicker) ensureScroll(visibleRows, total int) {
	if visibleRows < 1 {
		visibleRows = 1
	}
	if p.selected < p.scrollTop {
		p.scrollTop = p.selected
	}
	if p.selected >= p.scrollTop+visibleRows {
		p.scrollTop = p.selected - visibleRows + 1
	}
	if p.scrollTop < 0 {
		p.scrollTop = 0
	}
	maxScroll := total - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollTop > maxScroll {
		p.scrollTop = maxScroll
	}
}

func (p *serverPicker) clampSelection(visibleRows int) {
	filtered := p.filtered()
	if len(filtered) == 0 {
		p.selected = 0
		p.scrollTop = 0
		return
	}
	if p.selected >= len(filtered) {
		p.selected = len(filtered) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
	p.ensureScroll(visibleRows, len(filtered))
}

func (p *serverPicker) moveSelection(delta, visibleRows int) {
	filtered := p.filtered()
	if len(filtered) == 0 {
		return
	}
	p.selected += delta
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= len(filtered) {
		p.selected = len(filtered) - 1
	}
	p.ensureScroll(visibleRows, len(filtered))
}

func (p *serverPicker) setSearchWidth(tableWidth int) {
	if tableWidth > 12 {
		p.search.Width = tableWidth
	}
}

func (p *serverPicker) view(title string, dialogWidth, listHeight int, loading bool, progress string) string {
	cols := serverPickerColumnLayout()
	filtered := p.filtered()

	header := labelStyle.Render(title)
	hint := hintStyle.Render("type to filter · j/k navigate · enter select · r refresh · esc cancel")
	if loading {
		hint = hintStyle.Render("esc cancel")
	}
	searchRow := p.search.View()
	colHeader := renderServerPickerHeader(cols)

	var rows []string
	switch {
	case loading && len(p.entries) == 0:
		rows = append(rows, hintStyle.Render("loading servers…"))
	case len(p.entries) == 0:
		rows = append(rows, hintStyle.Render("no servers available"))
	case len(filtered) == 0:
		rows = append(rows, hintStyle.Render("no matching servers"))
	default:
		p.ensureScroll(listHeight, len(filtered))
		end := p.scrollTop + listHeight
		if end > len(filtered) {
			end = len(filtered)
		}
		for i := p.scrollTop; i < end; i++ {
			rows = append(rows, renderServerPickerRow(filtered[i], cols, i == p.selected))
		}
	}

	body := colHeader
	if len(rows) > 0 {
		body += "\n" + strings.Join(rows, "\n")
	}

	sections := []string{header, hint}
	if progress != "" {
		sections = append(sections, "", progress)
	}
	sections = append(sections, "", searchRow, "", body)
	return strings.Join(sections, "\n")
}

func (m Model) rebuildServerPicker() Model {
	dialogWidth, listHeight := m.serverPickerDialogSize()
	searchText := ""
	selectedName := ""
	if m.serverPickerOpen {
		searchText = m.serverPicker.search.Value()
		selectedName, _ = m.serverPicker.selectedName()
	}
	if selectedName == "" {
		if policy, ok := m.activePolicyFromStack(); ok {
			selectedName = m.policyRequestServer(policy.PolicyNumber)
		}
	}
	m.serverPicker = newServerPicker(dialogWidth, listHeight, m.servers, m.serversLoading, searchText, selectedName)
	return m
}

func (m Model) openRequestServerPicker() (Model, tea.Cmd) {
	if !m.onRunRequestPanel() {
		return m, nil
	}
	m, cmd := m.ensureServersLoaded()
	m = m.rebuildServerPicker()
	m.serverPickerOpen = true
	return m, tea.Batch(cmd, textinput.Blink)
}

func (m Model) refreshServerPicker() Model {
	if !m.serverPickerOpen {
		return m
	}
	return m.rebuildServerPicker()
}

func (m Model) closeServerPicker() Model {
	m.serverPickerOpen = false
	m.serverPicker.search.Blur()
	return m
}

func (m Model) handleServerPickerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	_, listHeight := m.serverPickerDialogSize()

	switch msg.String() {
	case "esc":
		if strings.TrimSpace(m.serverPicker.search.Value()) != "" {
			m.serverPicker.search.SetValue("")
			m.serverPicker.clampSelection(listHeight)
			return m, nil
		}
		return m.closeServerPicker(), nil
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if strings.TrimSpace(m.serverPicker.search.Value()) == "" {
			return m, tea.Quit
		}
	case "enter", "l", "right":
		name, ok := m.serverPicker.selectedName()
		if !ok {
			return m, nil
		}
		policy, ok := m.activePolicyFromStack()
		if !ok {
			return m.closeServerPicker(), nil
		}
		m = m.applyRequestServer(policy.PolicyNumber, name)
		return m.closeServerPicker(), nil
	case "up", "k", "ctrl+p":
		m.serverPicker.moveSelection(-1, listHeight)
		return m, nil
	case "down", "j", "ctrl+n":
		m.serverPicker.moveSelection(1, listHeight)
		return m, nil
	case "pgup", "u", "ctrl+u":
		m.serverPicker.moveSelection(-listHeight, listHeight)
		return m, nil
	case "pgdown", "d", "f", "ctrl+d":
		m.serverPicker.moveSelection(listHeight, listHeight)
		return m, nil
	case "r":
		if strings.TrimSpace(m.serverPicker.search.Value()) == "" && !m.serversLoading {
			return m.refreshServers()
		}
	}

	old := m.serverPicker.search.Value()
	var cmd tea.Cmd
	m.serverPicker.search, cmd = m.serverPicker.search.Update(msg)
	if m.serverPicker.search.Value() != old {
		m.serverPicker.selected = 0
		m.serverPicker.scrollTop = 0
		m.serverPicker.clampSelection(listHeight)
	}
	return m, cmd
}

func (m Model) serverPickerProgressView(tableWidth int) string {
	if !m.serversLoading {
		return ""
	}
	barWidth := tableWidth
	if barWidth < 12 {
		barWidth = 12
	}
	m.serverProgress.Width = barWidth
	label := m.status
	if label == "" {
		label = "refreshing servers…"
	}
	return labelStyle.Render(strings.ToLower(label)) + "\n" + m.serverProgress.View()
}

func (m Model) renderServerPickerDialog() string {
	dialogWidth, listHeight := m.serverPickerDialogSize()
	cols := serverPickerColumnLayout()
	m.serverPicker.setSearchWidth(cols.totalWidth())
	progress := m.serverPickerProgressView(cols.totalWidth())
	content := m.serverPicker.view("target server", dialogWidth, listHeight, m.serversLoading, progress)
	return modalDialogStyle.Width(dialogWidth).Render(content)
}

package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

const sidebarWidth = 22

type listItem struct {
	item MenuItem
}

func (i listItem) Title() string {
	title := i.item.Title
	if i.item.HasChildren() {
		return title + chevronStyle.Render(" ›")
	}
	return title
}
func (i listItem) Description() string { return i.item.Description }
func (i listItem) FilterValue() string { return i.item.Title }

type sidebarPanel struct {
	title string
	list  list.Model
	items []MenuItem
}

func newSidebarPanel(title string, items []MenuItem, height int) sidebarPanel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = listNormalTitleStyle
	delegate.Styles.DimmedTitle = listDimmedTitleStyle
	delegate.Styles.SelectedTitle = listSelectedTitleStyle
	delegate.Styles.FilterMatch = listFilterMatchStyle

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = listItem{item: item}
	}

	keyMap := list.DefaultKeyMap()
	keyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "b", "u", "ctrl+u"))
	keyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "f", "d", "ctrl+d"))

	l := list.New(listItems, delegate, sidebarWidth-4, height-6)
	l.Title = title
	l.KeyMap = keyMap
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return sidebarPanel{
		title: title,
		list:  l,
		items: items,
	}
}

func (p sidebarPanel) selectedItem() (MenuItem, bool) {
	selected := p.list.SelectedItem()
	if selected == nil {
		return MenuItem{}, false
	}
	li, ok := selected.(listItem)
	if !ok {
		return MenuItem{}, false
	}
	return li.item, true
}

func (p sidebarPanel) setIndex(i int) sidebarPanel {
	p.list.Select(i)
	return p
}

func (p sidebarPanel) withItems(items []MenuItem, height int) sidebarPanel {
	selectedTitle := ""
	if item, ok := p.selectedItem(); ok {
		selectedTitle = item.Title
	}

	next := newSidebarPanel(p.title, items, height)
	if selectedTitle == "" {
		return next
	}
	for i, item := range items {
		if item.Title == selectedTitle {
			return next.setIndex(i)
		}
	}
	return next
}

func (p sidebarPanel) view() string {
	return p.list.View()
}

func (p sidebarPanel) filtering() bool {
	return p.list.SettingFilter() || p.list.FilterState() == list.FilterApplied
}

func (p sidebarPanel) beginTypingFilter(msg tea.KeyMsg) (sidebarPanel, tea.Cmd) {
	if p.list.FilterState() == list.Unfiltered {
		p.list.SetFilterText(string(msg.Runes))
		return p, nil
	}

	if p.list.FilterState() == list.FilterApplied {
		p.list.SetFilterState(list.Filtering)
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func isVimMotionKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "j", "k", "h", "l", "g", "G",
		"u", "d", "b", "f",
		"up", "down", "left", "right",
		"pgup", "pgdown", "home", "end",
		"ctrl+u", "ctrl+d":
		return true
	}
	return false
}

func isTypingFilterKey(msg tea.KeyMsg) bool {
	if isVimMotionKey(msg) {
		return false
	}
	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 {
		return false
	}
	r := msg.Runes[0]
	if !unicode.IsPrint(r) {
		return false
	}
	switch r {
	case '/':
		return false
	}
	return true
}

const headerHeight = 1

func panelIndexAt(x int, collapsed bool, panelCount int) int {
	if collapsed {
		if x < sidebarWidth {
			return panelCount - 1
		}
		return -1
	}
	return x / sidebarWidth
}

func itemIndexAt(y, barLines int) int {
	return y - barLines - headerHeight
}

func formatBreadcrumb(panels []sidebarPanel) string {
	if len(panels) == 0 {
		return ""
	}

	var parts []string
	for i, p := range panels {
		if i == len(panels)-1 {
			parts = append(parts, pathCurrentStyle.Render(p.title))
		} else {
			parts = append(parts, pathCrumbStyle.Render(p.title))
		}
	}
	return strings.Join(parts, pathSepStyle.Render(" › "))
}

func renderPathBar(width int, panels []sidebarPanel) string {
	if width < 1 {
		width = 1
	}
	label := pathLabelStyle.Render(" nautlius ")
	sep := pathSepStyle.Render(" › ")
	path := formatBreadcrumb(panels)
	line := label + sep + path
	return pathBarStyle.Width(width).Render(line)
}

func pathBarLines(width int, panels []sidebarPanel) int {
	return lipgloss.Height(renderPathBar(width, panels))
}

func renderPolicyField(label, value string) string {
	return policyDetailLabelStyle.Render(label) +
		policyDetailSepStyle.Render(" · ") +
		policyDetailValueStyle.Render(value)
}

func renderPolicyDetailBar(width int, item MenuItem) string {
	if width < 1 {
		width = 1
	}

	title := emptyIconStyle.Render("◈") + " " + policyDetailTitleStyle.Render(item.Title)
	sep := policyDetailSepStyle.Render("  │  ")
	fields := renderPolicyField("number", item.PolicyNumber) +
		sep +
		renderPolicyField("control", item.ControlNumber) +
		sep +
		renderPolicyField("renewal", item.RenewalDate)

	line := title + sep + fields
	return policyDetailBarStyle.Width(width).Render(line)
}

func policyDetailBarLines(width int, item MenuItem) int {
	return lipgloss.Height(renderPolicyDetailBar(width, item))
}

func placeholderMainPane(width, height int, status string) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}

	var sections []string
	sections = append(sections,
		emptyIconStyle.Render("◈")+"  "+hintStyle.Render("select an option to begin"),
	)

	if status != "" {
		sections = append(sections,
			labelStyle.Render("last action"),
			statusStyle.Render(status),
		)
	}

	content := strings.Join(sections, "\n\n")
	return mainPaneStyle.Width(width).Height(height).Render(content)
}

func renderFooter(width int, showRefresh bool) string {
	if width < 1 {
		width = 1
	}
	seg := func(key, label string) string {
		return keyStyle.Render(key) + footerStyle.Render(" "+label)
	}
	parts := []string{
		seg("↑↓/jk", "navigate"),
		footerSepStyle.Render(" · "),
		seg("/", "filter"),
		footerSepStyle.Render(" · "),
		seg("enter/l", "open"),
		footerSepStyle.Render(" · "),
		seg("←/h", "back"),
	}
	if showRefresh {
		parts = append(parts,
			footerSepStyle.Render(" · "),
			seg("r", "refresh"),
		)
	}
	parts = append(parts,
		footerSepStyle.Render(" · "),
		seg("space", "commands"),
		footerSepStyle.Render(" · "),
		seg("q", "quit"),
	)
	line := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return footerStyle.Width(width).Render(line)
}

func footerLines(width int) int {
	return lipgloss.Height(renderFooter(width, false))
}

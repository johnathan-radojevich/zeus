package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	useTestListServers()
}

func update(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func apply(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func loadServersInto(m Model) Model {
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	return update(m, fetchServersCmd()())
}

func TestDrillDownAndBack(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadServersInto(m)

	if m.PanelCount() != 2 {
		t.Fatalf("expected 2 panels after selecting server utilities, got %d", m.PanelCount())
	}
	if m.panels[1].title != "server utilities" {
		t.Fatalf("expected second panel title server utilities, got %q", m.panels[1].title)
	}
	if len(m.servers) != services.DemoServerCount {
		t.Fatalf("expected %d cached servers, got %d", services.DemoServerCount, len(m.servers))
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.PanelCount() != 3 {
		t.Fatalf("expected 3 panels after selecting server, got %d", m.PanelCount())
	}
	if m.panels[2].title != "prod-us-east-1" {
		t.Fatalf("expected third panel title prod-us-east-1, got %q", m.panels[2].title)
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.PanelCount() != 4 {
		t.Fatalf("expected 4 panels after selecting actions, got %d", m.PanelCount())
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.PanelCount() != 3 {
		t.Fatalf("expected 3 panels after back, got %d", m.PanelCount())
	}
}

func TestLeafAction(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadServersInto(m)
	for range 2 {
		m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Status() == "" {
		t.Fatal("expected status message after leaf selection")
	}
	if m.PanelCount() != 4 {
		t.Fatalf("expected panel stack unchanged on leaf, got %d panels", m.PanelCount())
	}
}

func TestTypingFilter(t *testing.T) {
	panel := newSidebarPanel("home", ServerMenu(), 24)

	panel, _ = panel.beginTypingFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	panel, _ = panel.beginTypingFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	panel, _ = panel.beginTypingFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	if panel.list.FilterValue() != "dev" {
		t.Fatalf("expected filter value dev, got %q", panel.list.FilterValue())
	}

	item, ok := panel.selectedItem()
	if !ok || item.Title != "dev-local" {
		t.Fatalf("expected selected item dev-local, got ok=%v title=%q", ok, item.Title)
	}
}

func TestFilterPerPanel(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadServersInto(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if m.panels[1].list.FilterState() == list.Unfiltered {
		t.Fatal("expected second panel filter to be active after typing")
	}
}

func TestRootMenu(t *testing.T) {
	root := RootMenu(nil, false)
	if len(root) != 4 {
		t.Fatalf("expected 4 top-level items, got %d", len(root))
	}
	if root[0].Title != "server utilities" {
		t.Fatalf("expected server utilities first, got %q", root[0].Title)
	}
	if len(root[0].Children) != 1 || root[0].Children[0].Title != "no servers" {
		t.Fatalf("expected empty server placeholder, got %v", root[0].Children)
	}
	if root[1].Title != "story/spike management" {
		t.Fatalf("expected story/spike management second, got %q", root[1].Title)
	}
	if root[2].Title != "policy tools" {
		t.Fatalf("expected policy tools third, got %q", root[2].Title)
	}
	if root[3].Title != "codebase navigation" {
		t.Fatalf("expected codebase navigation fourth, got %q", root[3].Title)
	}
}

func TestPolicyDetailBar(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // policy tools
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // policies
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // view all

	view := m.View()
	if !strings.Contains(view, "POL-1042") {
		t.Fatalf("expected policy number in view, got %q", view[:min(200, len(view))])
	}
	if !strings.Contains(view, "employee access") {
		t.Fatalf("expected policy title in view, got %q", view[:min(200, len(view))])
	}
	if !strings.Contains(view, "2026-09-15") {
		t.Fatalf("expected renewal date in view, got %q", view[:min(200, len(view))])
	}
	if !strings.Contains(view, "CTRL-AC-042") {
		t.Fatalf("expected control number in view, got %q", view[:min(200, len(view))])
	}
}

func TestFindPolicy(t *testing.T) {
	policy, ok := FindPolicy("POL-2087")
	if !ok || policy.Title != "customer records" {
		t.Fatalf("expected customer records by policy number, got ok=%v title=%q", ok, policy.Title)
	}

	policy, ok = FindPolicy("ctrl-nw-022")
	if !ok || policy.Title != "east-west traffic" {
		t.Fatalf("expected east-west traffic by control number, got ok=%v title=%q", ok, policy.Title)
	}

	_, ok = FindPolicy("POL-9999")
	if ok {
		t.Fatal("expected no match for unknown policy number")
	}
}

func TestCommandMenuOpens(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeySpace})
	if !m.commandActive() {
		t.Fatal("expected command menu to open on space")
	}

	view := m.View()
	if !strings.Contains(view, "search policy") {
		t.Fatalf("expected search policy command in view, got %q", view[:min(300, len(view))])
	}
}

func TestPolicySearchByControlNumber(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = update(m, tea.KeyMsg{Type: tea.KeySpace})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("CTRL-DR-091")})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.commandActive() {
		t.Fatal("expected command menu to close after successful search")
	}
	if m.PanelCount() != 4 {
		t.Fatalf("expected 4 panels after policy search, got %d", m.PanelCount())
	}
	if m.CurrentSelection() != "audit logs" {
		t.Fatalf("expected audit logs selected, got %q", m.CurrentSelection())
	}
}

func TestServerListLoadsOnOpen(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected async fetch when opening server utilities")
	}
	if !m.serversLoading {
		t.Fatal("expected loading flag while fetch is in flight")
	}

	view := m.View()
	if !strings.Contains(view, "█") && !strings.Contains(view, "░") {
		t.Fatalf("expected progress bar in view while loading, got %q", view[:min(400, len(view))])
	}

	m = update(m, fetchServersCmd()())
	if len(m.servers) != services.DemoServerCount {
		t.Fatalf("expected %d servers after load, got %d", services.DemoServerCount, len(m.servers))
	}
	if m.serversLoading {
		t.Fatal("expected loading flag cleared after load")
	}
	if m.panels[1].title != "server utilities" {
		t.Fatal("expected server utilities panel to remain open")
	}
}

func TestServerListUsesCache(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadServersInto(m)
	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected cached servers to skip fetch while fresh")
	}
	if len(m.servers) != services.DemoServerCount {
		t.Fatalf("expected cached servers, got %d", len(m.servers))
	}
}

func TestServerListRefresh(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadServersInto(m)

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected refresh command on r in server utilities panel")
	}
	if !m.serversLoading {
		t.Fatal("expected loading flag during refresh")
	}

	view := m.View()
	if !strings.Contains(view, "refreshing servers") {
		t.Fatalf("expected refreshing status in view, got %q", view[:min(300, len(view))])
	}

	m = update(m, fetchServersCmd()())
	if len(m.servers) != services.DemoServerCount {
		t.Fatalf("expected %d servers after refresh, got %d", services.DemoServerCount, len(m.servers))
	}
}

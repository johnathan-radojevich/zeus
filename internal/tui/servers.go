package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	serverUtilitiesTitle = "server utilities"
	serverCacheTTL       = 5 * time.Minute
)

var listServers = services.ListServers

type serversLoadedMsg struct {
	items []MenuItem
	err   error
}

func resetListServers() {
	listServers = services.ListServers
}

func useTestListServers() {
	setListServersForTest(func(ctx context.Context) ([]services.Server, error) {
		return services.DefaultServers(), nil
	})
}

func setListServersForTest(fn func(context.Context) ([]services.Server, error)) {
	listServers = fn
}

func serversToMenuItems(servers []services.Server) []MenuItem {
	items := make([]MenuItem, len(servers))
	for i, s := range servers {
		desc := s.Environment
		if s.Region != "" {
			desc += " · " + s.Region
		}
		items[i] = server(s.Name, desc)
	}
	return items
}

func serverUtilitiesChildren(servers []MenuItem, loading bool) []MenuItem {
	switch {
	case loading && len(servers) == 0:
		return []MenuItem{{Title: "loading…", Description: "fetching server list"}}
	case len(servers) == 0:
		return []MenuItem{{Title: "no servers", Description: "press r to reload"}}
	default:
		return servers
	}
}

func (m Model) serversStale() bool {
	if m.serversFetchedAt.IsZero() {
		return true
	}
	return time.Since(m.serversFetchedAt) > serverCacheTTL
}

func (m Model) onServerUtilitiesPanel() bool {
	if len(m.panels) == 0 {
		return false
	}
	return m.panels[len(m.panels)-1].title == serverUtilitiesTitle
}

func (m Model) requestServers(force bool) (Model, tea.Cmd) {
	if m.serversLoading {
		return m, nil
	}
	if !force && len(m.servers) > 0 && !m.serversStale() {
		return m, nil
	}

	m.serversLoading = true
	if len(m.servers) == 0 {
		m.status = "loading servers…"
	} else {
		m.status = "refreshing servers…"
	}
	m = m.syncServerPanels()
	m, progressCmd := m.startServerProgress()
	return m, tea.Batch(fetchServersCmd(), progressCmd)
}

func fetchServersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		servers, err := listServers(ctx)
		if err != nil {
			return serversLoadedMsg{err: err}
		}
		return serversLoadedMsg{items: serversToMenuItems(servers)}
	}
}

func (m Model) handleServersLoaded(msg serversLoadedMsg) (Model, tea.Cmd) {
	m.serversLoading = false
	if msg.err != nil {
		if len(m.servers) == 0 {
			m.status = "failed to load servers: " + msg.err.Error()
		} else {
			m.status = "refresh failed — showing cached servers"
		}
		return m.syncServerPanels(), m.serverProgress.SetPercent(0)
	}

	m.servers = msg.items
	m.serversFetchedAt = time.Now()
	m.status = fmt.Sprintf("loaded %d servers", len(m.servers))
	m = m.syncServerPanels()
	if m.onRunRequestPanel() {
		m = m.syncRunRequestPanel()
		if m.serverPickerOpen {
			m = m.refreshServerPicker()
		}
		if policy, ok := m.activePolicyFromStack(); ok {
			m = m.ensureDefaultRequestServer(policy.PolicyNumber)
		}
	}
	return m, m.serverProgress.SetPercent(1)
}

func (m Model) syncServerPanels() Model {
	if len(m.panels) == 0 {
		return m
	}

	height := m.contentHeight()
	m.panels[0] = m.panels[0].withItems(m.rootMenu(), height)

	for i, panel := range m.panels {
		if panel.title == serverUtilitiesTitle {
			children := serverUtilitiesChildren(m.servers, m.serversLoading)
			m.panels[i] = panel.withItems(children, height)
		}
	}
	return m.resizePanels()
}

func (m Model) rootMenu() []MenuItem {
	return RootMenu(m.servers, m.serversLoading)
}

func (m Model) ensureServersLoaded() (Model, tea.Cmd) {
	m = m.syncServerPanels()
	return m.requestServers(false)
}

func (m Model) refreshServers() (Model, tea.Cmd) {
	return m.requestServers(true)
}

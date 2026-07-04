package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type serverProgressTickMsg struct{}

func newServerProgress(width int) progress.Model {
	full := "30"
	empty := "250"
	if lipgloss.HasDarkBackground() {
		full = "86"
		empty = "238"
	}

	p := progress.New(
		progress.WithWidth(width),
		progress.WithSolidFill(full),
		progress.WithFillCharacters('█', '░'),
	)
	p.EmptyColor = empty
	p.PercentageStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "243"})
	return p
}

func serverProgressTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return serverProgressTickMsg{}
	})
}

func (m Model) mainPaneBarWidth() int {
	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	mainWidth := innerWidth - m.visibleSidebarWidth()
	if mainWidth < 20 {
		mainWidth = 20
	}
	barWidth := mainWidth - 6
	if barWidth < 12 {
		barWidth = 12
	}
	return barWidth
}

func (m Model) startServerProgress() (Model, tea.Cmd) {
	m.serverProgress = newServerProgress(m.mainPaneBarWidth())
	return m, tea.Batch(m.serverProgress.SetPercent(0), serverProgressTickCmd())
}

func (m Model) handleServerProgressTick() (Model, tea.Cmd) {
	if m.serversLoading {
		if m.serverProgress.Percent() >= 0.92 {
			return m, serverProgressTickCmd()
		}
		return m, tea.Batch(m.serverProgress.IncrPercent(0.08), serverProgressTickCmd())
	}
	if m.policyEndpointsLoading || m.policyRequestRunning || m.policyRequestBodyLoading {
		return m.handlePolicyRequestProgressTick()
	}
	return m, nil
}

func (m Model) renderMainPane(width, height int) string {
	if m.serversLoading && !m.serverPickerOpen {
		m.serverProgress.Width = m.mainPaneBarWidth()
		return renderLoadingMainPane(width, height, m.serverProgress, m.status)
	}
	if m.onRunRequestPanel() {
		showingRequestPane := m.policyRequestResult != nil || m.policyRequestSelectedEndpoint != ""
		if m.policyEndpointsLoading && !showingRequestPane {
			m.policyRequestProgress.Width = m.mainPaneBarWidth()
			return renderLoadingMainPane(width, height, m.policyRequestProgress, m.status)
		}
		if showingRequestPane {
			return m.renderPolicyRequestPane(width, height)
		}
		if m.policyRequestErr != "" {
			return placeholderMainPane(width, height, "request failed: "+m.policyRequestErr)
		}
		return m.renderPolicyRunRequestSetupPane(width, height)
	}
	switch m.codebaseMode {
	case codebaseRuleKeyInput:
		return renderRuleKeySearchPane(width, height, m.ruleKeyInput)
	case codebaseClassInput:
		return renderClassSearchPane(width, height, m.classSearchInput)
	case codebaseClassResults:
		return renderClassResultsPane(width, height, m.classSearchResults)
	case codebaseXMLEditor:
		width, height := m.mainPaneEditorSize()
		return renderXMLEditorPane(width, height, m.xmlEditorPath, m.xmlEditor)
	}
	return placeholderMainPane(width, height, m.status)
}

func renderLoadingMainPane(width, height int, bar progress.Model, label string) string {
	if width < 10 {
		width = 10
	}
	if height < 6 {
		height = 6
	}
	if label == "" {
		label = "loading…"
	}

	sections := []string{
		labelStyle.Render(strings.ToLower(label)),
		bar.View(),
		hintStyle.Render("fetching from source"),
	}
	content := strings.Join(sections, "\n\n")
	return mainPaneStyle.Width(width).Height(height).Render(content)
}

package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func openPolicyRunRequest(m Model) Model {
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // policy tools
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // policies
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // view all
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter}) // employee access
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	return update(m, tea.KeyMsg{Type: tea.KeyEnter}) // run request
}

func loadRunRequestPanel(m Model) Model {
	m = openPolicyRunRequest(m)
	m = update(m, fetchPolicyEndpointsCmd("POL-1042")())
	return update(m, fetchServersCmd()())
}

func selectRunRequestEndpoint(m Model) Model {
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	return m
}

func selectRequestServer(m Model, serverName string) Model {
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	for _, r := range serverName {
		m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	return m
}

func finishPolicyRequest(m Model, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) Model {
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		m = update(m, cmd())
	}
	return update(m, runPolicyRequestCmd(policyNumber, endpointID, txn, serverName)())
}

func TestPolicyHasRunRequestOption(t *testing.T) {
	policies := accessControlPolicies()
	if len(policies[0].Children) != 2 {
		t.Fatalf("expected policy children, got %d", len(policies[0].Children))
	}
	if policies[0].Children[1].Title != runRequestTitle {
		t.Fatalf("expected run request option, got %q", policies[0].Children[1].Title)
	}
}

func TestPolicyRunRequestLoadsEndpoints(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = openPolicyRunRequest(m)
	if !m.onRunRequestPanel() {
		t.Fatal("expected run request panel")
	}

	m = update(m, fetchPolicyEndpointsCmd("POL-1042")())
	if len(m.policyEndpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(m.policyEndpoints))
	}
}

func TestPolicyRunRequestClearsOnNavigateBack(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setRunPolicyRequestForTest(func(ctx context.Context, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) (services.PolicyRequestResult, error) {
		return services.PolicyRequestResult{
			EndpointName: "validate rule",
			StatusCode:   200,
			ResponseBody: `{"status":"ok"}`,
		}, nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	if m.policyRequestRunning {
		t.Fatal("expected endpoint select to prepare request without running")
	}
	if m.policyRequestSelectedEndpoint != "validate" {
		t.Fatalf("expected validate endpoint selected, got %q", m.policyRequestSelectedEndpoint)
	}
	m = finishPolicyRequest(m, "POL-1042", "validate", services.PolicyTransactionNewBusiness, "prod-us-east-1")

	if m.policyRequestResult == nil {
		t.Fatal("expected result before navigating away")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.policyRequestResult != nil {
		t.Fatal("expected result cleared after leaving run request panel")
	}
	if m.onRunRequestPanel() {
		t.Fatal("expected to leave run request panel")
	}

	view := m.View()
	if strings.Contains(view, `"status":"ok"`) {
		t.Fatal("expected result view to be gone after navigating back")
	}
}

func TestPolicyRunRequestDisplaysResult(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setRunPolicyRequestForTest(func(ctx context.Context, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) (services.PolicyRequestResult, error) {
		return services.PolicyRequestResult{
			EndpointName: "validate rule",
			StatusCode:   200,
			RequestHeaders: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer test-token"},
			},
			RequestBody:  `{"policyNumber":"POL-1042"}`,
			ResponseBody: `{"status":"ok"}`,
			TargetURL:    "/api/validate",
		}, nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	if m.policyRequestRunning {
		t.Fatal("expected endpoint select to prepare request without running")
	}
	if m.policyRequestSelectedEndpoint != "validate" {
		t.Fatalf("expected validate endpoint selected, got %q", m.policyRequestSelectedEndpoint)
	}
	m = finishPolicyRequest(m, "POL-1042", "validate", services.PolicyTransactionNewBusiness, "prod-us-east-1")

	if m.policyRequestResult == nil {
		t.Fatal("expected policy request result")
	}
	if !strings.Contains(m.policyRequestResult.ResponseBody, `"status":"ok"`) {
		t.Fatalf("unexpected response body: %q", m.policyRequestResult.ResponseBody)
	}
	if !strings.Contains(m.Status(), "validate rule") {
		t.Fatalf("expected status with endpoint name, got %q", m.Status())
	}
	if m.policyRequestResult.RequestHeaders.Get("Authorization") != "Bearer test-token" {
		t.Fatal("expected request headers on result")
	}
	if !strings.Contains(m.policyRequestHeadersEditor.Value(), "Authorization: Bearer test-token") {
		t.Fatal("expected editable headers editor to be populated")
	}
	if !strings.Contains(m.policyRequestBodyEditor.Value(), "POL-1042") {
		t.Fatal("expected editable body editor to be populated")
	}

	width, height := m.mainPaneSize()
	rendered := m.renderPolicyRequestPane(width, height)
	if !strings.Contains(rendered, "headers") || !strings.Contains(rendered, "body") {
		t.Fatalf("expected sectioned request pane, got %q", rendered[:min(400, len(rendered))])
	}
}

func TestEndpointSelectPreparesWithoutRunning(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)

	if m.policyRequestRunning {
		t.Fatal("expected request not to run on endpoint select")
	}
	if m.policyRequestResult != nil {
		t.Fatal("expected no result until r is pressed")
	}
	if m.policyRequestSelectedEndpoint != "validate" {
		t.Fatalf("expected validate endpoint, got %q", m.policyRequestSelectedEndpoint)
	}

	view := m.View()
	if !strings.Contains(view, "press r to run") {
		t.Fatal("expected draft pane to prompt for r to run")
	}
	if !strings.Contains(view, " draft ") {
		t.Fatal("expected draft status badge")
	}
}

func TestRunRequestPanelListsEndpointsOnly(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)

	if m.CurrentSelection() != "validate rule" {
		t.Fatalf("expected first endpoint selected, got %q", m.CurrentSelection())
	}
	if len(m.panels[len(m.panels)-1].items) != 3 {
		t.Fatalf("expected endpoints only, got %d items", len(m.panels[len(m.panels)-1].items))
	}

	view := m.View()
	if !strings.Contains(view, "transaction type") {
		t.Fatal("expected setup pane to show transaction type")
	}
	if !strings.Contains(view, "target server") {
		t.Fatal("expected setup pane to show target server")
	}
}

func TestRequestServerSelection(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRequestServer(m, "staging")
	m = selectRunRequestEndpoint(m)

	if m.policyRequestServer("POL-1042") != "staging" {
		t.Fatalf("expected staging server, got %q", m.policyRequestServer("POL-1042"))
	}
	if !strings.Contains(m.policyRequestHeadersEditor.Value(), "X-Target-Server: staging") {
		t.Fatalf("expected staging header, got %q", m.policyRequestHeadersEditor.Value())
	}
	if !strings.Contains(m.policyRequestDraftTargetURL, "staging.internal") {
		t.Fatalf("expected staging target URL, got %q", m.policyRequestDraftTargetURL)
	}
}

func TestServerPickerRefresh(t *testing.T) {
	loads := 0
	setListServersForTest(func(ctx context.Context) ([]services.Server, error) {
		loads++
		return services.DefaultServers(), nil
	})
	t.Cleanup(useTestListServers)

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m.servers = serversToMenuItems(services.DefaultServers())
	m.serversFetchedAt = time.Now()
	m.serverPickerOpen = true
	m.serverPicker = newServerPicker(80, 12, m.servers, false, "", "")

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.serversLoading {
		t.Fatal("expected refresh to start loading")
	}
	if cmd == nil {
		t.Fatal("expected fetch command")
	}
	if !m.serverPickerOpen {
		t.Fatal("expected server modal to stay open")
	}

	m = update(m, fetchServersCmd()())
	if loads != 1 {
		t.Fatalf("expected one server fetch, got %d", loads)
	}
	if m.serversLoading {
		t.Fatal("expected loading to finish")
	}

	view := m.renderServerPickerDialog()
	if !strings.Contains(view, "r refresh") {
		t.Fatal("expected refresh hint in server modal")
	}
}

func TestServerPickerRefreshShowsProgress(t *testing.T) {
	setListServersForTest(func(ctx context.Context) ([]services.Server, error) {
		return services.DefaultServers(), nil
	})
	t.Cleanup(useTestListServers)

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m.servers = serversToMenuItems(services.DefaultServers())
	m.serversFetchedAt = time.Now()
	m.serverPickerOpen = true
	m.serverPicker = newServerPicker(80, 12, m.servers, false, "", "")

	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.serversLoading {
		t.Fatal("expected refresh to start loading")
	}

	view := m.renderServerPickerDialog()
	if !strings.Contains(view, "█") && !strings.Contains(view, "░") {
		t.Fatalf("expected progress bar in server modal, got %q", view[:min(400, len(view))])
	}
	if !strings.Contains(view, "refreshing servers") {
		t.Fatalf("expected refresh status in modal, got %q", view[:min(400, len(view))])
	}
}

func TestServerPickerSearchFilter(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m.servers = serversToMenuItems(services.DefaultServers())
	m.serverPickerOpen = true
	m.serverPicker = newServerPicker(80, 12, m.servers, false, "", "prod-us-east-1")

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f', 'l', 'e', 'e', 't', '-', '0', '4', '2'}})
	filtered := m.serverPicker.filtered()
	if len(filtered) != 1 || filtered[0].name != "fleet-042" {
		t.Fatalf("expected fleet-042 match, got %v", filtered)
	}

	view := m.renderServerPickerDialog()
	if !strings.Contains(view, "fleet-042") {
		t.Fatal("expected filtered server in dialog")
	}
	if strings.Contains(view, "prod-us-east-1") {
		t.Fatal("expected non-matching servers hidden from table")
	}
}

func TestTransactionTypeUpdatesHeaders(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setRunPolicyRequestForTest(func(ctx context.Context, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) (services.PolicyRequestResult, error) {
		return services.RunPolicyRequest(ctx, policyNumber, endpointID, txn, serverName)
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = m.applyTransactionType("POL-1042", services.PolicyTransactionEndorsement)
	m = selectRunRequestEndpoint(m)

	if m.policyRequestHeadersEditor.Value() == "" {
		t.Fatal("expected headers editor content")
	}
	if !strings.Contains(m.policyRequestHeadersEditor.Value(), "X-Transaction-Type: endorsement") {
		t.Fatalf("expected endorsement headers, got %q", m.policyRequestHeadersEditor.Value())
	}
	if !strings.Contains(m.policyRequestHeadersEditor.Value(), "X-Policy-Transaction: ENDR") {
		t.Fatal("expected ENDR policy transaction header")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !strings.Contains(m.policyRequestHeadersEditor.Value(), "X-Transaction-Type: new-business") {
		t.Fatalf("expected headers to update after switching type, got %q", m.policyRequestHeadersEditor.Value())
	}
}

func TestResponseViewportSearchAndScroll(t *testing.T) {
	longResponse := strings.Join([]string{
		`{"status":"ok"}`,
		`{"note":"alpha beta gamma"}`,
		`{"note":"second chunk"}`,
		`{"note":"beta appears again"}`,
		`{"tail":"done"}`,
	}, "\n")

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m = openPolicyRunRequest(m)
	m.policyRequestResult = &services.PolicyRequestResult{
		EndpointName: "validate rule",
		StatusCode:   200,
		ResponseBody: longResponse,
		TargetURL:    "/api/validate",
	}
	m = m.openPolicyRequestEditors(*m.policyRequestResult)

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if m.policyRequestEditorFocus != policyRequestFocusResponse {
		t.Fatal("expected response focus")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.policyRequestResponseSearchActive {
		t.Fatal("expected response search to activate")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b', 'e', 't', 'a'}})
	if len(findMatchLines(m.policyRequestResponseRaw, m.policyRequestResponseSearch.Value())) < 2 {
		t.Fatal("expected multiple beta matches")
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.policyRequestResponseMatchIndex != 1 {
		t.Fatalf("expected second match selected, got index %d", m.policyRequestResponseMatchIndex)
	}

	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.policyRequestResponseViewport.YOffset < 1 {
		t.Fatal("expected viewport to scroll down")
	}
}

func TestPolicyRequestEditorFocusSwitch(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setRunPolicyRequestForTest(func(ctx context.Context, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) (services.PolicyRequestResult, error) {
		return services.PolicyRequestResult{
			EndpointName:   "validate rule",
			StatusCode:     200,
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}},
			RequestBody:    `{}`,
			ResponseBody:   `{}`,
		}, nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	if m.policyRequestRunning {
		t.Fatal("expected endpoint select to prepare request without running")
	}
	if m.policyRequestSelectedEndpoint != "validate" {
		t.Fatalf("expected validate endpoint selected, got %q", m.policyRequestSelectedEndpoint)
	}
	m = finishPolicyRequest(m, "POL-1042", "validate", services.PolicyTransactionNewBusiness, "prod-us-east-1")

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if m.policyRequestEditorFocus != policyRequestFocusBody {
		t.Fatal("expected body editor focus after pressing 2")
	}
	if !m.policyRequestBodyEditor.Focused() {
		t.Fatal("expected body editor to be focused")
	}
}

func TestPolicyRequestSidebarWorksWhenUnfocused(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setRunPolicyRequestForTest(func(ctx context.Context, policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) (services.PolicyRequestResult, error) {
		return services.PolicyRequestResult{
			EndpointName:   "validate rule",
			StatusCode:     200,
			RequestHeaders: map[string][]string{"Content-Type": {"application/json"}},
			RequestBody:    `{}`,
			ResponseBody:   `{}`,
		}, nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	if m.policyRequestRunning {
		t.Fatal("expected endpoint select to prepare request without running")
	}
	if m.policyRequestSelectedEndpoint != "validate" {
		t.Fatalf("expected validate endpoint selected, got %q", m.policyRequestSelectedEndpoint)
	}
	m = finishPolicyRequest(m, "POL-1042", "validate", services.PolicyTransactionNewBusiness, "prod-us-east-1")

	first := m.CurrentSelection()
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.CurrentSelection() == first {
		t.Fatal("expected sidebar j to work while request pane is unfocused")
	}
}

func TestPolicyRequestPaneShowsProgressWhileRunning(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m.policyRequestSelectedEndpoint = "validate"
	m.policyRequestDraftEndpointName = "validate rule"
	m.policyRequestRunning = true
	m.status = "running request on staging…"
	m, _ = m.startPolicyRequestProgress()
	m = m.openPolicyRequestDraft("Content-Type: application/json", `{"policyNumber":"POL-1042"}`)

	width, height := m.mainPaneSize()
	rendered := m.renderPolicyRequestPane(width, height)
	if !strings.Contains(rendered, "headers") || !strings.Contains(rendered, "body") {
		t.Fatal("expected request sections to remain visible while running")
	}
	if !strings.Contains(rendered, "running request") {
		t.Fatal("expected running status in progress area")
	}
}

func TestPolicyRequestBodyRefreshFromSource(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	setFetchPolicyRequestBodyForTest(func(ctx context.Context, policyNumber string, endpoint *services.PolicyEndpoint, txn services.PolicyTransactionType) (string, error) {
		return `{"refreshed":true,"policyNumber":"` + policyNumber + `"}`, nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	originalBody := m.policyRequestBodyEditor.Value()

	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if cmd == nil {
		t.Fatal("expected body refresh command")
	}
	if !m.policyRequestBodyLoading {
		t.Fatal("expected body refresh to start loading")
	}

	m = update(m, fetchPolicyRequestBodyCmd("POL-1042", "validate", services.PolicyTransactionNewBusiness)())
	if m.policyRequestBodyLoading {
		t.Fatal("expected body refresh to finish loading")
	}
	if m.policyRequestBodyEditor.Value() == originalBody {
		t.Fatal("expected body editor to update after refresh")
	}
	if !strings.Contains(m.policyRequestBodyEditor.Value(), `"refreshed":true`) {
		t.Fatalf("expected refreshed body, got %q", m.policyRequestBodyEditor.Value())
	}
	if !strings.Contains(m.Status(), "body refreshed from source") {
		t.Fatalf("expected refresh status, got %q", m.Status())
	}
}

func TestPolicyRequestExportBruno(t *testing.T) {
	setListPolicyEndpointsForTest(func(ctx context.Context, policyNumber string) ([]services.PolicyEndpoint, error) {
		return services.DefaultPolicyEndpoints(policyNumber), nil
	})
	var copied string
	setCopyToClipboardForTest(func(text string) error {
		copied = text
		return nil
	})
	defer resetPolicyRequests()

	m := New()
	m.width = 120
	m.height = 40
	m.ready = true

	m = loadRunRequestPanel(m)
	m = selectRunRequestEndpoint(m)
	m = selectRequestServer(m, "staging")

	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !strings.Contains(m.Status(), "copied Bruno request to clipboard") {
		t.Fatalf("expected export status, got %q", m.Status())
	}
	if !strings.Contains(copied, "name: validate rule") {
		t.Fatalf("expected bruno export, got %q", copied)
	}
	if !strings.Contains(copied, "staging.internal") {
		t.Fatalf("expected target URL in export, got %q", copied)
	}
	if !strings.Contains(copied, "POL-1042") {
		t.Fatal("expected exported body to include policy number")
	}
}

func TestPolicyRequestPaneFitsHeight(t *testing.T) {
	m := New()
	m.width = 120
	m.height = 40
	m.ready = true
	m.policyRequestResult = &services.PolicyRequestResult{
		EndpointName:   "validate rule",
		StatusCode:     200,
		RequestHeaders: map[string][]string{"Accept": {"application/json"}},
		RequestBody:    "{\n  \"ok\": true\n}",
		ResponseBody:   "{\n  \"status\": \"ok\"\n}",
		TargetURL:      "/api/validate",
	}
	m = m.openPolicyRequestEditors(*m.policyRequestResult)

	width, height := m.mainPaneSize()
	rendered := m.renderPolicyRequestPane(width, height)
	renderedLines := lipgloss.Height(rendered)
	if renderedLines > height+2 {
		t.Fatalf("pane taller than available space: rendered %d lines, height %d", renderedLines, height)
	}
	if !strings.Contains(rendered, "headers") || !strings.Contains(rendered, "response") {
		t.Fatal("expected all sections in rendered pane")
	}
}

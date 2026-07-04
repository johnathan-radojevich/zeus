package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/radojevich/zeus/internal/services"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"
)

const runRequestTitle = "run request"

const (
	transactionNewBusinessTitle = "new business"
	transactionEndorsementTitle = "endorsement"
)

var (
	listPolicyEndpoints    = services.ListPolicyEndpoints
	runPolicyRequest       = services.RunPolicyRequest
	fetchPolicyRequestBody = services.FetchPolicyRequestBody
	copyToClipboard        = defaultCopyToClipboard
)

func defaultCopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

type policyEndpointsLoadedMsg struct {
	policyNumber string
	items        []MenuItem
	err          error
}

type policyRequestResultMsg struct {
	policyNumber string
	endpointID   string
	result       services.PolicyRequestResult
	err          error
}

type policyRequestBodyLoadedMsg struct {
	policyNumber string
	endpointID   string
	body         string
	err          error
}

func setListPolicyEndpointsForTest(fn func(context.Context, string) ([]services.PolicyEndpoint, error)) {
	listPolicyEndpoints = fn
}

func setRunPolicyRequestForTest(fn func(context.Context, string, string, services.PolicyTransactionType, string) (services.PolicyRequestResult, error)) {
	runPolicyRequest = fn
}

func setFetchPolicyRequestBodyForTest(fn func(context.Context, string, *services.PolicyEndpoint, services.PolicyTransactionType) (string, error)) {
	fetchPolicyRequestBody = fn
}

func setCopyToClipboardForTest(fn func(string) error) {
	copyToClipboard = fn
}

func resetPolicyRequests() {
	listPolicyEndpoints = services.ListPolicyEndpoints
	runPolicyRequest = services.RunPolicyRequest
	fetchPolicyRequestBody = services.FetchPolicyRequestBody
	copyToClipboard = defaultCopyToClipboard
}

func endpointsToMenuItems(endpoints []services.PolicyEndpoint) []MenuItem {
	items := make([]MenuItem, len(endpoints))
	for i, e := range endpoints {
		items[i] = MenuItem{
			Title:       e.Title,
			Description: e.Description,
			EndpointID:  e.ID,
		}
	}
	return items
}

func policyEndpointChildren(endpoints []MenuItem, loading bool) []MenuItem {
	switch {
	case loading && len(endpoints) == 0:
		return []MenuItem{{Title: "loading…", Description: "fetching endpoints"}}
	case len(endpoints) == 0:
		return []MenuItem{{Title: "no endpoints", Description: "unavailable"}}
	default:
		return endpoints
	}
}

func runRequestPanelChildren(endpoints []MenuItem, loading bool) []MenuItem {
	return policyEndpointChildren(endpoints, loading)
}

func (m Model) policyTransactionType(policyNumber string) services.PolicyTransactionType {
	if m.policyTransactionTypes == nil {
		return services.PolicyTransactionNewBusiness
	}
	if txn, ok := m.policyTransactionTypes[policyNumber]; ok {
		return txn
	}
	return services.PolicyTransactionNewBusiness
}

func (m Model) setPolicyTransactionType(policyNumber string, txn services.PolicyTransactionType) Model {
	if m.policyTransactionTypes == nil {
		m.policyTransactionTypes = make(map[string]services.PolicyTransactionType)
	}
	m.policyTransactionTypes[policyNumber] = txn
	m.status = "transaction type · " + txn.Label()
	return m
}

func (m Model) policyRequestServer(policyNumber string) string {
	if m.policyRequestServers != nil {
		if name, ok := m.policyRequestServers[policyNumber]; ok && name != "" {
			return name
		}
	}
	if len(m.servers) > 0 {
		return m.servers[0].Title
	}
	return ""
}

func (m Model) ensureDefaultRequestServer(policyNumber string) Model {
	if m.policyRequestServer(policyNumber) == "" {
		return m
	}
	if m.policyRequestServers == nil {
		m.policyRequestServers = make(map[string]string)
	}
	if _, ok := m.policyRequestServers[policyNumber]; !ok {
		m.policyRequestServers[policyNumber] = m.policyRequestServer(policyNumber)
	}
	return m
}

func (m Model) applyRequestServer(policyNumber, serverName string) Model {
	if m.policyRequestServers == nil {
		m.policyRequestServers = make(map[string]string)
	}
	m.policyRequestServers[policyNumber] = serverName
	m.status = "target server · " + serverName
	if m.policyRequestSelectedEndpoint != "" {
		m = m.refreshPolicyRequestDraftTarget()
		m = m.refreshPolicyRequestHeadersEditor()
	}
	return m
}

func (m Model) refreshPolicyRequestDraftTarget() Model {
	if m.policyRequestSelectedEndpoint == "" {
		return m
	}
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return m
	}
	ep, ok := services.FindPolicyEndpoint(policy.PolicyNumber, m.policyRequestSelectedEndpoint)
	if !ok {
		return m
	}
	m.policyRequestDraftTargetURL = services.ResolvePolicyTargetURL(
		m.policyRequestServer(policy.PolicyNumber),
		ep.TargetURL,
	)
	return m
}

func (m Model) refreshPolicyRequestHeadersEditor() Model {
	endpointID := ""
	if m.policyRequestResult != nil {
		endpointID = m.policyRequestResult.EndpointID
	} else if m.policyRequestSelectedEndpoint != "" {
		endpointID = m.policyRequestSelectedEndpoint
	} else {
		return m
	}
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return m
	}
	headers := services.BuildPolicyRequestHeaders(services.PolicyRequestHeaderInput{
		PolicyNumber:    policy.PolicyNumber,
		EndpointID:      endpointID,
		TransactionType: m.policyTransactionType(policy.PolicyNumber),
		ServerName:      m.policyRequestServer(policy.PolicyNumber),
	})
	m.policyRequestHeadersEditor.SetValue(services.FormatHTTPHeaders(headers))
	return m
}

func (m Model) refreshPolicyRequestDraftBody() Model {
	if m.policyRequestSelectedEndpoint == "" || m.policyRequestResult != nil {
		return m
	}
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return m
	}
	ep, ok := services.FindPolicyEndpoint(policy.PolicyNumber, m.policyRequestSelectedEndpoint)
	if !ok {
		return m
	}
	body := services.BuildPolicyRequestBody(
		policy.PolicyNumber,
		&ep,
		m.policyTransactionType(policy.PolicyNumber),
	)
	m.policyRequestBodyEditor.SetValue(body)
	return m
}

func (m Model) refreshPolicyRequestBodyFromSource() (Model, tea.Cmd) {
	if m.policyRequestSelectedEndpoint == "" || m.policyRequestResult != nil ||
		m.policyRequestBodyLoading || m.policyRequestRunning {
		return m, nil
	}
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return m, nil
	}

	m.policyRequestBodyLoading = true
	m.status = "refreshing body from source…"
	m, progressCmd := m.startPolicyRequestProgress()
	txn := m.policyTransactionType(policy.PolicyNumber)
	return m, tea.Batch(
		fetchPolicyRequestBodyCmd(policy.PolicyNumber, m.policyRequestSelectedEndpoint, txn),
		progressCmd,
	)
}

func fetchPolicyRequestBodyCmd(policyNumber, endpointID string, txn services.PolicyTransactionType) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		ep, ok := services.FindPolicyEndpoint(policyNumber, endpointID)
		if !ok {
			return policyRequestBodyLoadedMsg{
				policyNumber: policyNumber,
				endpointID:   endpointID,
				err:          fmt.Errorf("unknown endpoint %q", endpointID),
			}
		}

		body, err := fetchPolicyRequestBody(ctx, policyNumber, &ep, txn)
		return policyRequestBodyLoadedMsg{
			policyNumber: policyNumber,
			endpointID:   endpointID,
			body:         body,
			err:          err,
		}
	}
}

func (m Model) handlePolicyRequestBodyLoaded(msg policyRequestBodyLoadedMsg) (Model, tea.Cmd) {
	m.policyRequestBodyLoading = false

	policy, ok := m.activePolicyFromStack()
	if !ok || policy.PolicyNumber != msg.policyNumber ||
		m.policyRequestSelectedEndpoint != msg.endpointID {
		return m, m.policyRequestProgress.SetPercent(0)
	}
	if msg.err != nil {
		m.status = "body refresh failed: " + msg.err.Error()
		return m, m.policyRequestProgress.SetPercent(0)
	}

	m.policyRequestBodyEditor.SetValue(msg.body)
	m.status = "body refreshed from source"
	return m, m.policyRequestProgress.SetPercent(1)
}

func (m Model) applyTransactionType(policyNumber string, txn services.PolicyTransactionType) Model {
	m = m.setPolicyTransactionType(policyNumber, txn)
	m = m.syncRunRequestPanel()
	if m.policyRequestResult != nil {
		m = m.refreshPolicyRequestHeadersEditor()
	} else if m.policyRequestSelectedEndpoint != "" {
		m = m.refreshPolicyRequestHeadersEditor()
		m = m.refreshPolicyRequestDraftBody()
		m = m.refreshPolicyRequestDraftTarget()
	}
	return m
}

func (m MenuItem) IsEndpoint() bool {
	return m.EndpointID != ""
}

func (m Model) activePolicyFromStack() (MenuItem, bool) {
	for i := len(m.panels) - 1; i >= 0; i-- {
		item, ok := m.panels[i].selectedItem()
		if ok && item.IsPolicy() {
			return item, true
		}
	}
	return MenuItem{}, false
}

func (m Model) onRunRequestPanel() bool {
	if len(m.panels) == 0 {
		return false
	}
	return m.panels[len(m.panels)-1].title == runRequestTitle
}

func (m Model) clearPolicyRequestDisplay() Model {
	m.policyRequestRunning = false
	m.policyEndpointsLoading = false
	m.policyRequestResult = nil
	m.policyRequestErr = ""
	m.policyRequestSelectedEndpoint = ""
	m.policyRequestDraftEndpointName = ""
	m.policyRequestDraftTargetURL = ""
	m = m.blurPolicyRequestEditors()
	return m
}

func (m Model) syncPolicyRequestDisplay() Model {
	if !m.onRunRequestPanel() {
		return m.clearPolicyRequestDisplay()
	}
	return m
}

func (m Model) ensurePolicyEndpoints(policy MenuItem) (Model, tea.Cmd) {
	if m.policyEndpointsLoading && m.policyEndpointsPolicy == policy.PolicyNumber {
		return m, nil
	}
	if len(m.policyEndpoints) > 0 && m.policyEndpointsPolicy == policy.PolicyNumber && !m.policyEndpointsLoading {
		return m, nil
	}

	m.policyEndpointsPolicy = policy.PolicyNumber
	m.policyEndpointsLoading = true
	m.status = "loading endpoints…"
	m = m.syncRunRequestPanel()
	m, progressCmd := m.startPolicyRequestProgress()
	return m, tea.Batch(fetchPolicyEndpointsCmd(policy.PolicyNumber), progressCmd)
}

func fetchPolicyEndpointsCmd(policyNumber string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		endpoints, err := listPolicyEndpoints(ctx, policyNumber)
		if err != nil {
			return policyEndpointsLoadedMsg{policyNumber: policyNumber, err: err}
		}
		return policyEndpointsLoadedMsg{
			policyNumber: policyNumber,
			items:        endpointsToMenuItems(endpoints),
		}
	}
}

func (m Model) handlePolicyEndpointsLoaded(msg policyEndpointsLoadedMsg) (Model, tea.Cmd) {
	m.policyEndpointsLoading = false
	if msg.err != nil {
		m.status = "failed to load endpoints: " + msg.err.Error()
		return m.syncRunRequestPanel(), m.policyRequestProgress.SetPercent(0)
	}

	m.policyEndpoints = msg.items
	m.policyEndpointsPolicy = msg.policyNumber
	m.status = fmt.Sprintf("loaded %d endpoints", len(msg.items))
	m = m.syncRunRequestPanel()
	return m, m.policyRequestProgress.SetPercent(1)
}

func (m Model) syncRunRequestPanel() Model {
	for i, panel := range m.panels {
		if panel.title == runRequestTitle {
			children := runRequestPanelChildren(m.policyEndpoints, m.policyEndpointsLoading)
			m.panels[i] = panel.withItems(children, m.contentHeight())
		}
	}
	return m.resizePanels()
}

func (m Model) preparePolicyRequest(policy MenuItem, endpoint MenuItem) Model {
	m.policyRequestResult = nil
	m.policyRequestErr = ""
	m.policyRequestSelectedEndpoint = endpoint.EndpointID
	m.policyRequestDraftEndpointName = endpoint.Title

	ep, ok := services.FindPolicyEndpoint(policy.PolicyNumber, endpoint.EndpointID)
	if ok {
		m.policyRequestDraftTargetURL = services.ResolvePolicyTargetURL(
			m.policyRequestServer(policy.PolicyNumber),
			ep.TargetURL,
		)
	} else {
		m.policyRequestDraftTargetURL = ""
	}

	txn := m.policyTransactionType(policy.PolicyNumber)
	headers := services.BuildPolicyRequestHeaders(services.PolicyRequestHeaderInput{
		PolicyNumber:    policy.PolicyNumber,
		EndpointID:      endpoint.EndpointID,
		TransactionType: txn,
		ServerName:      m.policyRequestServer(policy.PolicyNumber),
	})
	body := services.BuildPolicyRequestBody(policy.PolicyNumber, &ep, txn)
	m = m.openPolicyRequestDraft(services.FormatHTTPHeaders(headers), body)
	m.status = fmt.Sprintf("ready · %s · r to run", endpoint.Title)
	return m
}

func (m Model) runSelectedPolicyRequest() (Model, tea.Cmd) {
	if m.policyRequestSelectedEndpoint == "" || m.policyRequestRunning || m.policyRequestBodyLoading {
		return m, nil
	}
	policy, ok := m.activePolicyFromStack()
	if !ok {
		return m, nil
	}
	server := m.policyRequestServer(policy.PolicyNumber)
	if server == "" {
		m.status = "select a target server before running"
		return m, nil
	}
	return m.startPolicyRequest(policy.PolicyNumber, m.policyRequestSelectedEndpoint, server)
}

func (m Model) startPolicyRequest(policyNumber, endpointID, serverName string) (Model, tea.Cmd) {
	if m.policyRequestRunning {
		return m, nil
	}

	m.policyRequestRunning = true
	m.policyRequestResult = nil
	m.policyRequestErr = ""
	m.status = fmt.Sprintf("running request on %s…", serverName)
	m, progressCmd := m.startPolicyRequestProgress()
	txn := m.policyTransactionType(policyNumber)
	return m, tea.Batch(runPolicyRequestCmd(policyNumber, endpointID, txn, serverName), progressCmd)
}

func runPolicyRequestCmd(policyNumber, endpointID string, txn services.PolicyTransactionType, serverName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := runPolicyRequest(ctx, policyNumber, endpointID, txn, serverName)
		return policyRequestResultMsg{
			policyNumber: policyNumber,
			endpointID:   endpointID,
			result:       result,
			err:          err,
		}
	}
}

func (m Model) handlePolicyRequestResult(msg policyRequestResultMsg) (Model, tea.Cmd) {
	m.policyRequestRunning = false
	if msg.err != nil {
		m.policyRequestErr = msg.err.Error()
		m.status = "request failed: " + msg.err.Error()
		return m, m.policyRequestProgress.SetPercent(0)
	}

	m.policyRequestResult = &msg.result
	m = m.openPolicyRequestEditors(msg.result)
	m = m.resizePolicyRequestEditors()
	m.status = fmt.Sprintf("%s · %s · %d", msg.result.EndpointName, msg.result.ServerName, msg.result.StatusCode)
	return m, m.policyRequestProgress.SetPercent(1)
}

func (m Model) startPolicyRequestProgress() (Model, tea.Cmd) {
	m.policyRequestProgress = newServerProgress(m.mainPaneBarWidth())
	return m, tea.Batch(m.policyRequestProgress.SetPercent(0), serverProgressTickCmd())
}

func (m Model) handlePolicyRequestProgressTick() (Model, tea.Cmd) {
	if m.policyRequestProgress.Percent() >= 0.92 {
		return m, serverProgressTickCmd()
	}
	if !m.policyRequestRunning && !m.policyRequestBodyLoading {
		return m, nil
	}
	return m, tea.Batch(m.policyRequestProgress.IncrPercent(0.08), serverProgressTickCmd())
}

func (m Model) buildBrunoExport() (string, error) {
	if m.policyRequestSelectedEndpoint == "" && m.policyRequestResult == nil {
		return "", fmt.Errorf("no request to export")
	}

	name := m.policyRequestDraftEndpointName
	url := m.policyRequestDraftTargetURL
	if m.policyRequestResult != nil {
		name = m.policyRequestResult.EndpointName
		url = m.policyRequestResult.TargetURL
	}
	if strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("request URL is empty")
	}

	headers, err := services.ParseHTTPHeaders(m.policyRequestHeadersEditor.Value())
	if err != nil {
		return "", fmt.Errorf("invalid headers: %w", err)
	}

	return services.FormatBrunoRequest(services.BrunoRequest{
		Name:    name,
		Method:  "POST",
		URL:     url,
		Headers: headers,
		Body:    m.policyRequestBodyEditor.Value(),
	}), nil
}

func (m Model) exportPolicyRequestBruno() Model {
	content, err := m.buildBrunoExport()
	if err != nil {
		m.status = "export failed: " + err.Error()
		return m
	}
	if err := copyToClipboard(content); err != nil {
		m.status = "clipboard unavailable: " + err.Error()
		return m
	}
	m.status = "copied Bruno request to clipboard"
	return m
}

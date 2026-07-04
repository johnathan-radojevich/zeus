package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// PolicyEndpoint is a query target for a policy run request.
type PolicyEndpoint struct {
	ID          string
	Title       string
	Description string
	RequestURL  string // source for the request payload
	TargetURL   string // endpoint to invoke with that payload
}

// PolicyRequestResult is the outcome of a run request.
type PolicyRequestResult struct {
	EndpointID      string
	EndpointName    string
	ServerName      string
	RequestURL      string
	TargetURL       string
	RequestHeaders  http.Header
	RequestBody     string
	ResponseBody    string
	StatusCode      int
}

// DefaultPolicyEndpoints returns demo endpoints for a policy.
func DefaultPolicyEndpoints(policyNumber string) []PolicyEndpoint {
	return []PolicyEndpoint{
		{
			ID:          "validate",
			Title:       "validate rule",
			Description: "POST /validate",
			RequestURL:  "/api/policies/" + policyNumber + "/request/validate",
			TargetURL:   "/api/policies/" + policyNumber + "/validate",
		},
		{
			ID:          "evaluate",
			Title:       "evaluate",
			Description: "POST /evaluate",
			RequestURL:  "/api/policies/" + policyNumber + "/request/evaluate",
			TargetURL:   "/api/policies/" + policyNumber + "/evaluate",
		},
		{
			ID:          "simulate",
			Title:       "simulate",
			Description: "POST /simulate",
			RequestURL:  "/api/policies/" + policyNumber + "/request/simulate",
			TargetURL:   "/api/policies/" + policyNumber + "/simulate",
		},
	}
}

// ListPolicyEndpoints returns endpoints available for a policy.
func ListPolicyEndpoints(ctx context.Context, policyNumber string) ([]PolicyEndpoint, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(200 * time.Millisecond):
		return DefaultPolicyEndpoints(policyNumber), nil
	}
}

// PolicyTransactionType controls how a policy request is classified.
type PolicyTransactionType string

const (
	PolicyTransactionNewBusiness PolicyTransactionType = "new-business"
	PolicyTransactionEndorsement PolicyTransactionType = "endorsement"
)

func (t PolicyTransactionType) Label() string {
	switch t {
	case PolicyTransactionEndorsement:
		return "endorsement"
	default:
		return "new business"
	}
}

// PolicyRequestHeaderInput builds outbound request headers for a run.
type PolicyRequestHeaderInput struct {
	PolicyNumber    string
	EndpointID      string
	TransactionType PolicyTransactionType
	ServerName      string
}

// BuildPolicyRequestHeaders returns HTTP headers for the given transaction type.
func BuildPolicyRequestHeaders(in PolicyRequestHeaderInput) http.Header {
	if in.TransactionType == "" {
		in.TransactionType = PolicyTransactionNewBusiness
	}

	requestID := fmt.Sprintf("req-%s-%s-%s-%d",
		in.PolicyNumber, in.EndpointID, in.TransactionType, time.Now().UnixNano())

	headers := http.Header{
		"Accept":          []string{"application/json"},
		"Content-Type":    []string{"application/json"},
		"Authorization":   []string{"Bearer demo-token"},
		"User-Agent":      []string{"nautlius/1.0"},
		"X-Policy-Number": []string{in.PolicyNumber},
		"X-Endpoint-Id":   []string{in.EndpointID},
		"X-Request-Id":    []string{requestID},
		"X-Transaction-Type": []string{string(in.TransactionType)},
	}

	switch in.TransactionType {
	case PolicyTransactionEndorsement:
		headers.Set("X-Policy-Transaction", "ENDR")
		headers.Set("X-Business-Context", "endorsement")
	default:
		headers.Set("X-Policy-Transaction", "NB")
		headers.Set("X-Business-Context", "new-business")
	}

	if in.ServerName != "" {
		headers.Set("X-Target-Server", in.ServerName)
	}

	return headers
}

// ResolvePolicyTargetURL builds the full URL for a request on a target server.
func ResolvePolicyTargetURL(serverName, path string) string {
	if server, ok := FindServer(serverName); ok && server.HostURL != "" {
		return JoinHostAndPath(server.HostURL, path)
	}
	if serverName == "" {
		return path
	}
	return JoinHostAndPath("https://"+serverName+".internal", path)
}

// RunPolicyRequest fetches a request payload then calls the target endpoint.
func RunPolicyRequest(ctx context.Context, policyNumber, endpointID string, txnType PolicyTransactionType, serverName string) (PolicyRequestResult, error) {
	endpoints := DefaultPolicyEndpoints(policyNumber)
	var endpoint *PolicyEndpoint
	for i := range endpoints {
		if endpoints[i].ID == endpointID {
			endpoint = &endpoints[i]
			break
		}
	}
	if endpoint == nil {
		return PolicyRequestResult{}, fmt.Errorf("unknown endpoint %q", endpointID)
	}

	select {
	case <-ctx.Done():
		return PolicyRequestResult{}, ctx.Err()
	case <-time.After(250 * time.Millisecond):
	}

	requestBody, err := FetchPolicyRequestBody(ctx, policyNumber, endpoint, txnType)
	if err != nil {
		return PolicyRequestResult{}, err
	}
	requestHeaders := BuildPolicyRequestHeaders(PolicyRequestHeaderInput{
		PolicyNumber:    policyNumber,
		EndpointID:      endpoint.ID,
		TransactionType: txnType,
		ServerName:      serverName,
	})

	select {
	case <-ctx.Done():
		return PolicyRequestResult{}, ctx.Err()
	case <-time.After(350 * time.Millisecond):
	}

	response := map[string]any{
		"policy":          policyNumber,
		"endpoint":        endpoint.ID,
		"server":          serverName,
		"transactionType": txnType,
		"status":          "ok",
		"decision":        "allow",
		"evaluated":       time.Now().UTC().Format(time.RFC3339),
		"requestKey":      endpoint.ID,
	}
	responseBytes, _ := json.MarshalIndent(response, "", "  ")

	return PolicyRequestResult{
		EndpointID:     endpoint.ID,
		EndpointName:   endpoint.Title,
		ServerName:     serverName,
		RequestURL:     endpoint.RequestURL,
		TargetURL:      ResolvePolicyTargetURL(serverName, endpoint.TargetURL),
		RequestHeaders: requestHeaders,
		RequestBody:    requestBody,
		ResponseBody:   string(responseBytes),
		StatusCode:     200,
	}, nil
}

// FormatHTTPHeaders renders headers in HTTP wire format, sorted by name.
func FormatHTTPHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	for _, key := range keys {
		for _, value := range headers[key] {
			lines = append(lines, fmt.Sprintf("%s: %s", key, value))
		}
	}
	return strings.Join(lines, "\n")
}

// FetchPolicyRequestBody loads a templated request body from the external source.
func FetchPolicyRequestBody(ctx context.Context, policyNumber string, endpoint *PolicyEndpoint, txnType PolicyTransactionType) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}

	return BuildPolicyRequestBody(policyNumber, endpoint, txnType), nil
}

// BuildPolicyRequestBody returns the JSON payload for a policy run request.
func BuildPolicyRequestBody(policyNumber string, endpoint *PolicyEndpoint, txnType PolicyTransactionType) string {
	if txnType == "" {
		txnType = PolicyTransactionNewBusiness
	}

	payload := map[string]any{
		"policyNumber":    policyNumber,
		"ruleKey":         endpoint.ID,
		"source":          endpoint.RequestURL,
		"transactionType": string(txnType),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func FindPolicyEndpoint(policyNumber, endpointID string) (PolicyEndpoint, bool) {
	for _, e := range DefaultPolicyEndpoints(policyNumber) {
		if e.ID == endpointID {
			return e, true
		}
	}
	return PolicyEndpoint{}, false
}

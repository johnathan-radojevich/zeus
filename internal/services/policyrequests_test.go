package services

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListPolicyEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	endpoints, err := ListPolicyEndpoints(ctx, "POL-1042")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}
}

func TestRunPolicyRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := RunPolicyRequest(ctx, "POL-1042", "validate", PolicyTransactionNewBusiness, "prod-us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", result.StatusCode)
	}
	if result.RequestBody == "" || result.ResponseBody == "" {
		t.Fatal("expected request and response bodies")
	}
	if result.RequestHeaders.Get("Content-Type") != "application/json" {
		t.Fatalf("expected content-type header, got %q", result.RequestHeaders.Get("Content-Type"))
	}
	if FormatHTTPHeaders(result.RequestHeaders) == "" {
		t.Fatal("expected formatted request headers")
	}
	if result.RequestHeaders.Get("X-Transaction-Type") != "new-business" {
		t.Fatalf("expected new-business transaction header, got %q", result.RequestHeaders.Get("X-Transaction-Type"))
	}
	if result.RequestHeaders.Get("X-Policy-Transaction") != "NB" {
		t.Fatalf("expected NB policy transaction header, got %q", result.RequestHeaders.Get("X-Policy-Transaction"))
	}
	if result.RequestHeaders.Get("X-Target-Server") != "prod-us-east-1" {
		t.Fatalf("expected target server header, got %q", result.RequestHeaders.Get("X-Target-Server"))
	}
	if !strings.Contains(result.TargetURL, "prod-us-east-1.internal") {
		t.Fatalf("expected resolved target URL, got %q", result.TargetURL)
	}
	if result.ServerName != "prod-us-east-1" {
		t.Fatalf("expected server on result, got %q", result.ServerName)
	}
}

func TestRunPolicyRequestEndorsementHeaders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := RunPolicyRequest(ctx, "POL-1042", "validate", PolicyTransactionEndorsement, "staging")
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestHeaders.Get("X-Transaction-Type") != "endorsement" {
		t.Fatalf("expected endorsement transaction header, got %q", result.RequestHeaders.Get("X-Transaction-Type"))
	}
	if result.RequestHeaders.Get("X-Policy-Transaction") != "ENDR" {
		t.Fatalf("expected ENDR policy transaction header, got %q", result.RequestHeaders.Get("X-Policy-Transaction"))
	}
	if !strings.Contains(result.RequestBody, `"transactionType": "endorsement"`) {
		t.Fatalf("expected endorsement in body, got %q", result.RequestBody)
	}
}

func TestFormatHTTPHeaders(t *testing.T) {
	formatted := FormatHTTPHeaders(http.Header{
		"Z-Last":  {"2"},
		"Accept":  {"application/json"},
		"X-Multi": {"a", "b"},
	})
	if !strings.Contains(formatted, "Accept: application/json") {
		t.Fatalf("unexpected format: %q", formatted)
	}
	if !strings.Contains(formatted, "X-Multi: a") || !strings.Contains(formatted, "X-Multi: b") {
		t.Fatalf("expected multiple header values, got %q", formatted)
	}
	if strings.Index(formatted, "Accept:") > strings.Index(formatted, "Z-Last:") {
		t.Fatalf("expected sorted headers, got %q", formatted)
	}
}

func TestRunPolicyRequestUnknownEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := RunPolicyRequest(ctx, "POL-1042", "missing", PolicyTransactionNewBusiness, "prod-us-east-1")
	if err == nil {
		t.Fatal("expected error for unknown endpoint")
	}
}

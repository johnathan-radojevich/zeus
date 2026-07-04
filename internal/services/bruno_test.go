package services

import (
	"net/http"
	"strings"
	"testing"
)

func TestFormatBrunoRequest(t *testing.T) {
	content := FormatBrunoRequest(BrunoRequest{
		Name:   "validate rule",
		Method: "POST",
		URL:    "https://prod-us-east-1.internal/api/policies/POL-1042/validate",
		Headers: http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer demo-token"},
		},
		Body: "{\n  \"policyNumber\": \"POL-1042\"\n}",
	})

	for _, want := range []string{
		"name: validate rule",
		"post {",
		"url: https://prod-us-east-1.internal/api/policies/POL-1042/validate",
		"body: json",
		"auth: none",
		"Content-Type: application/json",
		`Authorization: "Bearer demo-token"`,
		"body:json {",
		`"policyNumber": "POL-1042"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in bruno export:\n%s", want, content)
		}
	}
}

func TestFormatBrunoRequestWithoutBody(t *testing.T) {
	content := FormatBrunoRequest(BrunoRequest{
		Name:   "ping",
		Method: "POST",
		URL:    "https://example.internal/ping",
	})
	if !strings.Contains(content, "body: none") {
		t.Fatalf("expected body: none, got %q", content)
	}
	if strings.Contains(content, "body:json") {
		t.Fatal("expected no body block for empty payload")
	}
}

func TestParseHTTPHeaders(t *testing.T) {
	headers, err := ParseHTTPHeaders("Accept: application/json\nAuthorization: Bearer token\n")
	if err != nil {
		t.Fatal(err)
	}
	if headers.Get("Accept") != "application/json" {
		t.Fatalf("unexpected accept header: %q", headers.Get("Accept"))
	}
	if headers.Get("Authorization") != "Bearer token" {
		t.Fatalf("unexpected authorization header: %q", headers.Get("Authorization"))
	}
}

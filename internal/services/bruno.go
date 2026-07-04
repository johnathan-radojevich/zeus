package services

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// BrunoRequest holds the fields needed to render a Bruno .bru request file.
type BrunoRequest struct {
	Name    string
	Method  string
	URL     string
	Headers http.Header
	Body    string
}

// ParseHTTPHeaders parses HTTP wire-format header lines ("Name: value").
func ParseHTTPHeaders(text string) (http.Header, error) {
	headers := make(http.Header)
	for lineNum, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header on line %d", lineNum+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("empty header name on line %d", lineNum+1)
		}
		headers.Add(key, value)
	}
	return headers, nil
}

// FormatBrunoRequest renders a request as a Bruno-compatible .bru file.
func FormatBrunoRequest(req BrunoRequest) string {
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "post"
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "request"
	}

	body := strings.TrimRight(req.Body, "\n")
	bodyKind := "none"
	if body != "" {
		bodyKind = "json"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "meta {\n  name: %s\n  type: http\n  seq: 1\n}\n\n", name)
	fmt.Fprintf(&b, "%s {\n  url: %s\n  body: %s\n  auth: none\n}\n", method, req.URL, bodyKind)

	if len(req.Headers) > 0 {
		b.WriteString("\nheaders {\n")
		keys := make([]string, 0, len(req.Headers))
		for key := range req.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, value := range req.Headers[key] {
				fmt.Fprintf(&b, "  %s: %s\n", bruFormatKey(key), bruFormatValue(value))
			}
		}
		b.WriteString("}\n")
	}

	if body != "" {
		b.WriteString("\nbody:json {\n")
		b.WriteString(indentBrunoBlock(body))
		b.WriteString("\n}\n")
	}

	return b.String()
}

func bruFormatKey(key string) string {
	if bruNeedsQuotes(key) {
		return bruQuote(key)
	}
	return key
}

func bruFormatValue(value string) string {
	if bruNeedsQuotes(value) {
		return bruQuote(value)
	}
	return value
}

func bruNeedsQuotes(s string) bool {
	if s == "" {
		return true
	}
	return strings.ContainsAny(s, " \":{}") || strings.Contains(s, "\n") || strings.Contains(s, "\t")
}

func bruQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func indentBrunoBlock(body string) string {
	lines := strings.Split(body, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("  ")
		b.WriteString(line)
	}
	return b.String()
}

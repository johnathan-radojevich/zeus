package services

import (
	"context"
	"testing"
)

func TestFindServer(t *testing.T) {
	server, ok := FindServer("staging")
	if !ok {
		t.Fatal("expected to find staging")
	}
	if server.Environment != "pre-production" {
		t.Fatalf("unexpected environment: %q", server.Environment)
	}
	if server.HostURL != "https://staging.internal" {
		t.Fatalf("unexpected host URL: %q", server.HostURL)
	}
	if _, ok := FindServer("missing"); ok {
		t.Fatal("expected missing server lookup to fail")
	}
}

func TestListServersRespectsContext(t *testing.T) {
	fetchServersReportHTML = func(ctx context.Context, reportURL string) (string, error) {
		return "", ctx.Err()
	}
	t.Cleanup(func() {
		fetchServersReportHTML = defaultFetchServersReportHTML
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ListServers(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

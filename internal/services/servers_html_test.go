package services

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestParseServersHTML(t *testing.T) {
	htmlDoc, err := os.ReadFile("testdata/https_suputils.html")
	if err != nil {
		t.Fatal(err)
	}

	servers, err := ParseServersHTML(string(htmlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	first := servers[0]
	if first.Environment != "Development" {
		t.Fatalf("unexpected environment: %q", first.Environment)
	}
	if first.Name != "WAS8-suputils01" {
		t.Fatalf("unexpected server name: %q", first.Name)
	}
	if first.Region != "Dev" {
		t.Fatalf("unexpected region: %q", first.Region)
	}
	if first.HostURL != "https://suputils01.dev.allstate.com" {
		t.Fatalf("unexpected host URL: %q", first.HostURL)
	}

	second := servers[1]
	if second.HostURL != "https://suputils02.qa.allstate.com:8443" {
		t.Fatalf("unexpected host URL with port: %q", second.HostURL)
	}

	third := servers[2]
	if third.HostURL != "http://suputils03.preprod.allstate.com" {
		t.Fatalf("unexpected http host URL: %q", third.HostURL)
	}
}

func TestServerNameFromDashCell(t *testing.T) {
	if got := serverNameFromDashCell("Alliance-Dev-WAS8-suputils01"); got != "WAS8-suputils01" {
		t.Fatalf("unexpected server name: %q", got)
	}
}

func TestHostURLFromString(t *testing.T) {
	tests := map[string]string{
		"https://suputils01.dev.allstate.com/reports/app": "https://suputils01.dev.allstate.com",
		"suputils01.dev.allstate.com/path":               "https://suputils01.dev.allstate.com",
		"http://example.co.uk/admin":                     "http://example.co.uk",
	}
	for input, want := range tests {
		if got := baseURLFromString(input); got != want {
			t.Fatalf("baseURLFromString(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestListServersUsesReportHTML(t *testing.T) {
	fetchServersReportHTML = func(ctx context.Context, reportURL string) (string, error) {
		if reportURL != defaultServersReportURL {
			t.Fatalf("unexpected report URL: %q", reportURL)
		}
		return `<table><tr>
			<td>Development</td><td></td><td></td><td>Alliance-Dev-WAS8-suputils01</td><td>https://suputils01.dev.allstate.com/app</td><td></td>
		</tr></table>`, nil
	}
	t.Cleanup(func() {
		fetchServersReportHTML = defaultFetchServersReportHTML
		ResetServerFleet()
	})

	servers, err := ListServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].HostURL != "https://suputils01.dev.allstate.com" {
		t.Fatalf("unexpected host URL: %q", servers[0].HostURL)
	}
	found, ok := FindServer("WAS8-suputils01")
	if !ok {
		t.Fatal("expected loaded fleet to be searchable")
	}
	if found.HostURL != servers[0].HostURL {
		t.Fatalf("unexpected fleet lookup host URL: %q", found.HostURL)
	}
}

func TestJoinHostAndPath(t *testing.T) {
	got := JoinHostAndPath("https://suputils01.dev.allstate.com", "/api/policies/POL-1042/validate")
	want := "https://suputils01.dev.allstate.com/api/policies/POL-1042/validate"
	if got != want {
		t.Fatalf("unexpected joined URL: %q", got)
	}
}

func TestResolvePolicyTargetURLUsesServerHost(t *testing.T) {
	SetServerFleet([]Server{
		{Name: "WAS8-suputils01", HostURL: "https://suputils01.dev.allstate.com"},
	})
	t.Cleanup(ResetServerFleet)

	got := ResolvePolicyTargetURL("WAS8-suputils01", "/api/policies/POL-1042/validate")
	if !strings.Contains(got, "suputils01.dev.allstate.com") {
		t.Fatalf("expected report host in target URL, got %q", got)
	}
}

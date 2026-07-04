package services

import (
	"context"
	"fmt"
	"strings"
)

// DemoServerCount is the number of fake servers returned by DefaultServers.
const DemoServerCount = 100

const defaultServersReportURL = DefaultServersReportURL

// Server is a managed host in the fleet.
type Server struct {
	Name        string
	Environment string
	Region      string
	HostURL     string
}

var (
	demoServers = buildDemoServers()
	serverFleet []Server
)

var fetchServersReportHTML = defaultFetchServersReportHTML

func defaultFetchServersReportHTML(ctx context.Context, reportURL string) (string, error) {
	body, err := fetchServersReport(ctx, reportURL)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func buildDemoServers() []Server {
	servers := []Server{
		{Name: "prod-us-east-1", Environment: "production", Region: "us east", HostURL: "https://prod-us-east-1.internal"},
		{Name: "prod-eu-west-1", Environment: "production", Region: "eu west", HostURL: "https://prod-eu-west-1.internal"},
		{Name: "staging", Environment: "pre-production", Region: "global", HostURL: "https://staging.internal"},
		{Name: "dev-local", Environment: "local development", Region: "local", HostURL: "https://dev-local.internal"},
	}

	regions := []string{"us east", "us west", "eu west", "eu central", "ap south", "ap northeast", "local"}
	envs := []string{"production", "pre-production", "staging", "development", "qa"}

	for i := len(servers) + 1; i <= DemoServerCount; i++ {
		name := fmt.Sprintf("fleet-%03d", i)
		servers = append(servers, Server{
			Name:        name,
			Environment: envs[(i-1)%len(envs)],
			Region:      regions[(i-1)%len(regions)],
			HostURL:       "https://" + name + ".internal",
		})
	}
	return servers
}

// DefaultServers returns the demo fleet used in tests and offline flows.
func DefaultServers() []Server {
	out := make([]Server, len(demoServers))
	copy(out, demoServers)
	return out
}

// SetServerFleet replaces the active fleet, primarily for tests.
func SetServerFleet(servers []Server) {
	serverFleet = append([]Server(nil), servers...)
}

// ResetServerFleet clears the active fleet loaded from the report.
func ResetServerFleet() {
	serverFleet = nil
}

// FindServer returns a fleet member by name.
func FindServer(name string) (Server, bool) {
	for _, s := range serverFleet {
		if s.Name == name {
			return s, true
		}
	}
	for _, s := range demoServers {
		if s.Name == name {
			return s, true
		}
	}
	return Server{}, false
}

// ListServers fetches the current fleet from the support utilities report.
func ListServers(ctx context.Context) ([]Server, error) {
	htmlDoc, err := fetchServersReportHTML(ctx, defaultServersReportURL)
	if err != nil {
		return nil, err
	}

	servers, err := ParseServersHTML(htmlDoc)
	if err != nil {
		return nil, err
	}

	SetServerFleet(servers)
	return append([]Server(nil), servers...), nil
}

// JoinHostAndPath combines a server base URL with an endpoint path.
func JoinHostAndPath(hostURL, path string) string {
	hostURL = strings.TrimSpace(hostURL)
	path = strings.TrimSpace(path)
	if hostURL == "" {
		return path
	}
	if path == "" {
		return strings.TrimRight(hostURL, "/")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(hostURL, "/") + path
}

package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const DefaultServersReportURL = "https://semssupportutilities.allstate.com/reports/Alliance/nonprod/WAS8/Https_Suputils.html"

// ParseServersHTML extracts servers from the support utilities report table.
// Each data row is expected to have six cells; environment, server id, and URL
// are read from the first, fourth, and fifth cells respectively.
func ParseServersHTML(htmlDoc string) ([]Server, error) {
	root, err := html.Parse(strings.NewReader(htmlDoc))
	if err != nil {
		return nil, err
	}

	var servers []Server
	for _, tr := range htmlNodes(root, "tr") {
		cells := tableCells(tr)
		if len(cells) < 5 {
			continue
		}
		server, ok := serverFromReportRow(cells)
		if !ok {
			continue
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in report")
	}
	return servers, nil
}

func serverFromReportRow(cells []*html.Node) (Server, bool) {
	if len(cells) == 0 || cells[0].Data == "th" {
		return Server{}, false
	}
	environment := cellText(cells[0])
	name := serverNameFromDashCell(cellText(cells[3]))
	hostURL := hostURLFromCell(cells[4])
	if environment == "" || name == "" || hostURL == "" {
		return Server{}, false
	}
	return Server{
		Name:        name,
		Environment: environment,
		Region:      regionFromDashCell(cellText(cells[3])),
		HostURL:     hostURL,
	}, true
}

func serverNameFromDashCell(value string) string {
	parts := splitDashParts(value)
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "-" + parts[len(parts)-1]
	}
	return strings.TrimSpace(value)
}

func regionFromDashCell(value string) string {
	parts := splitDashParts(value)
	if len(parts) >= 3 {
		return parts[len(parts)-3]
	}
	return ""
}

func splitDashParts(value string) []string {
	raw := strings.Split(strings.TrimSpace(value), "-")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func hostURLFromCell(cell *html.Node) string {
	if href := firstAnchorHref(cell); href != "" {
		return baseURLFromString(href)
	}
	return baseURLFromString(cellText(cell))
}

func baseURLFromString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + parsed.Host
}

func firstAnchorHref(n *html.Node) string {
	for _, node := range htmlNodes(n, "a") {
		for _, attr := range node.Attr {
			if attr.Key == "href" && strings.TrimSpace(attr.Val) != "" {
				return strings.TrimSpace(attr.Val)
			}
		}
	}
	return ""
}

func cellText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func tableCells(tr *html.Node) []*html.Node {
	var cells []*html.Node
	for child := tr.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		switch child.Data {
		case "td", "th":
			cells = append(cells, child)
		}
	}
	return cells
}

func htmlNodes(root *html.Node, tag string) []*html.Node {
	var nodes []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			nodes = append(nodes, n)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return nodes
}

func fetchServersReport(ctx context.Context, reportURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reportURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server report returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

package tui

import (
	"strings"

	"github.com/radojevich/zeus/internal/services"
)

// MenuItem is a node in the sidebar navigation tree.
type MenuItem struct {
	Title        string
	Description  string
	Children     []MenuItem
	Action       string // shown in the main pane when a leaf is selected
	PolicyNumber  string // set for selectable policies
	ControlNumber string // control identifier for policies
	RenewalDate   string // renewal date for policies
	EndpointID    string // set for policy run request endpoints
}

func (m MenuItem) HasChildren() bool {
	return len(m.Children) > 0
}

func (m MenuItem) IsPolicy() bool {
	return m.PolicyNumber != ""
}

func policy(title, desc, number, control, renewal, action string) MenuItem {
	return MenuItem{
		Title:         title,
		Description:   desc,
		PolicyNumber:  number,
		ControlNumber: control,
		RenewalDate:   renewal,
		Children: []MenuItem{
			{Title: "open policy", Description: "view rule details", Action: action},
			{
				Title:       runRequestTitle,
				Description: "query policy endpoints",
				Children:    []MenuItem{{Title: "loading…", Description: "open to fetch endpoints"}},
			},
		},
	}
}

// FindPolicy locates a policy by policy number or control number.
func FindPolicy(query string) (MenuItem, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return MenuItem{}, false
	}
	for _, p := range allPolicies() {
		if strings.EqualFold(p.PolicyNumber, q) || strings.EqualFold(p.ControlNumber, q) {
			return p, true
		}
	}
	return MenuItem{}, false
}

// RootMenu returns the top-level app menu.
func RootMenu(servers []MenuItem, loading bool) []MenuItem {
	return []MenuItem{
		{
			Title:       "server utilities",
			Description: "manage servers and infrastructure",
			Children:    serverUtilitiesChildren(servers, loading),
		},
		{
			Title:       "story/spike management",
			Description: "plan stories and track spikes",
			Children:    StorySpikeMenu(),
		},
		{
			Title:       "policy tools",
			Description: "review and enforce policies",
			Children:    PolicyToolsMenu(),
		},
		{
			Title:       "codebase navigation",
			Description: "browse and jump through source",
			Children:    CodebaseMenu(),
		},
	}
}

// ServerMenu returns the static demo server list (used in tests and fallbacks).
func ServerMenu() []MenuItem {
	return serversToMenuItems(services.DefaultServers())
}

func server(name, desc string) MenuItem {
	return MenuItem{
		Title:       name,
		Description: desc,
		Children: []MenuItem{
			{
				Title:       "actions",
				Description: "run server operations",
				Children: []MenuItem{
					{Title: "deploy", Description: "ship a new version", Children: deployOptions()},
					{Title: "restart", Description: "rolling restart", Action: "restart queued for " + name},
					{Title: "scale", Description: "change replica count", Children: scaleOptions(name)},
					{Title: "shell", Description: "open remote shell", Action: "opening shell on " + name + "…"},
				},
			},
			{
				Title:       "observe",
				Description: "logs and metrics",
				Children: []MenuItem{
					{Title: "logs", Description: "tail application logs", Children: logOptions(name)},
					{Title: "metrics", Description: "cpu, memory, latency", Children: metricOptions(name)},
					{Title: "events", Description: "recent cluster events", Action: "fetching events for " + name + "…"},
				},
			},
			{
				Title:       "configure",
				Description: "environment and secrets",
				Children: []MenuItem{
					{Title: "environment", Description: "env vars", Children: envOptions(name)},
					{Title: "secrets", Description: "manage credentials", Action: "opening secrets vault for " + name},
					{Title: "networking", Description: "ports and ingress", Children: networkOptions(name)},
				},
			},
		},
	}
}

// StorySpikeMenu returns story and spike management options.
func StorySpikeMenu() []MenuItem {
	return []MenuItem{
		{
			Title:       "stories",
			Description: "sprint stories and backlog",
			Children: []MenuItem{
				{Title: "create story", Description: "add a new story", Action: "opening story creator…"},
				{Title: "view backlog", Description: "browse unscheduled work", Children: storyBacklog()},
				{Title: "active sprint", Description: "current sprint board", Children: activeSprint()},
			},
		},
		{
			Title:       "spikes",
			Description: "time-boxed explorations",
			Children: []MenuItem{
				{Title: "new spike", Description: "start a spike", Action: "creating spike…"},
				{Title: "active spikes", Description: "in-progress spikes", Children: activeSpikes()},
				{Title: "spike archive", Description: "completed spikes", Action: "loading spike archive…"},
			},
		},
	}
}

func storyBacklog() []MenuItem {
	return []MenuItem{
		{Title: "auth refactor", Description: "priority: high", Action: "opening auth refactor…"},
		{Title: "api rate limits", Description: "priority: medium", Action: "opening api rate limits…"},
		{Title: "dashboard widgets", Description: "priority: low", Action: "opening dashboard widgets…"},
	}
}

func activeSprint() []MenuItem {
	return []MenuItem{
		{Title: "checkout flow", Description: "in progress", Action: "opening checkout flow…"},
		{Title: "email notifications", Description: "in review", Action: "opening email notifications…"},
		{Title: "search indexing", Description: "blocked", Action: "opening search indexing…"},
	}
}

func activeSpikes() []MenuItem {
	return []MenuItem{
		{Title: "graphql caching", Description: "2 days left", Action: "opening graphql caching spike…"},
		{Title: "wasm prototype", Description: "4 days left", Action: "opening wasm prototype spike…"},
	}
}

// PolicyToolsMenu returns policy review and enforcement options.
func PolicyToolsMenu() []MenuItem {
	return []MenuItem{
		{
			Title:       "policies",
			Description: "browse and edit policy rules",
			Children: []MenuItem{
				{Title: "view all", Description: "list active policies", Children: allPolicies()},
				{Title: "create policy", Description: "define a new rule", Action: "opening policy editor…"},
				{Title: "by category", Description: "filter by type", Children: policyCategories()},
			},
		},
		{
			Title:       "compliance",
			Description: "audit and reporting",
			Children: []MenuItem{
				{Title: "run audit", Description: "check current compliance", Action: "starting compliance audit…"},
				{Title: "violations", Description: "open policy violations", Children: policyViolations()},
				{Title: "export report", Description: "download compliance summary", Action: "generating compliance report…"},
			},
		},
		{
			Title:       "enforcement",
			Description: "apply and simulate policy actions",
			Children: []MenuItem{
				{Title: "dry run", Description: "simulate without applying", Action: "running policy dry run…"},
				{Title: "apply changes", Description: "enforce pending updates", Action: "applying policy changes…"},
				{Title: "rollback", Description: "revert last enforcement", Action: "rolling back policy enforcement…"},
			},
		},
	}
}

// CodebaseMenu returns source navigation options for the repo.
func CodebaseMenu() []MenuItem {
	return []MenuItem{
		{
			Title:       findXMLForRuleKeyTitle,
			Description: "search repo for matching rule xml",
			Action:      ActionFindXMLForRuleKey,
		},
		{
			Title:       findImplementingClassTitle,
			Description: "search repo for class implementations",
			Action:      ActionFindImplementingClass,
		},
	}
}

func policyCategories() []MenuItem {
	return []MenuItem{
		{Title: "access control", Description: "iam and permissions", Children: accessControlPolicies()},
		{Title: "data retention", Description: "storage and deletion rules", Children: dataRetentionPolicies()},
		{Title: "network", Description: "firewall and ingress rules", Children: networkPolicies()},
	}
}

func allPolicies() []MenuItem {
	policies := append([]MenuItem{}, accessControlPolicies()...)
	policies = append(policies, dataRetentionPolicies()...)
	policies = append(policies, networkPolicies()...)
	return policies
}

func accessControlPolicies() []MenuItem {
	return []MenuItem{
		policy("employee access", "standard access controls", "POL-1042", "CTRL-AC-042", "2026-09-15", "opening employee access policy…"),
		policy("vendor access", "third-party account rules", "POL-1048", "CTRL-AC-048", "2026-06-30", "opening vendor access policy…"),
	}
}

func dataRetentionPolicies() []MenuItem {
	return []MenuItem{
		policy("customer records", "crm data lifecycle", "POL-2087", "CTRL-DR-087", "2026-12-01", "opening customer records policy…"),
		policy("audit logs", "log retention windows", "POL-2091", "CTRL-DR-091", "2027-03-15", "opening audit logs policy…"),
	}
}

func networkPolicies() []MenuItem {
	return []MenuItem{
		policy("ingress baseline", "public endpoint rules", "POL-3015", "CTRL-NW-015", "2026-08-20", "opening ingress baseline policy…"),
		policy("east-west traffic", "internal service mesh", "POL-3022", "CTRL-NW-022", "2026-11-10", "opening east-west traffic policy…"),
	}
}

func policyViolations() []MenuItem {
	return []MenuItem{
		{Title: "critical", Description: "requires immediate action", Action: "loading critical violations…"},
		{Title: "warning", Description: "review recommended", Action: "loading warning violations…"},
		{Title: "resolved", Description: "recently closed items", Action: "loading resolved violations…"},
	}
}

func deployOptions() []MenuItem {
	return []MenuItem{
		{Title: "rolling update", Description: "replace pods gradually", Action: "starting rolling deploy…"},
		{Title: "blue / green", Description: "swap traffic to new stack", Action: "preparing blue/green cutover…"},
		{Title: "canary", Description: "route 5% traffic first", Action: "launching canary (5%)…"},
		{Title: "rollback", Description: "revert to previous release", Action: "rolling back to last stable…"},
	}
}

func scaleOptions(server string) []MenuItem {
	return []MenuItem{
		{Title: "scale up (+1)", Description: "add one replica", Action: "scaling " + server + " up by 1"},
		{Title: "scale down (-1)", Description: "remove one replica", Action: "scaling " + server + " down by 1"},
		{Title: "autoscale", Description: "edit hpa settings", Children: []MenuItem{
			{Title: "min replicas", Description: "floor for autoscaling", Action: "editing min replicas…"},
			{Title: "max replicas", Description: "ceiling for autoscaling", Action: "editing max replicas…"},
			{Title: "target cpu", Description: "utilization threshold", Action: "editing cpu target…"},
		}},
	}
}

func logOptions(server string) []MenuItem {
	return []MenuItem{
		{Title: "tail (live)", Description: "stream new log lines", Action: "tailing logs on " + server + "…"},
		{Title: "last hour", Description: "recent history", Action: "loading last hour of logs…"},
		{Title: "errors only", Description: "filter level ≥ error", Action: "filtering error logs…"},
	}
}

func metricOptions(server string) []MenuItem {
	return []MenuItem{
		{Title: "cpu", Description: "utilization over time", Action: "loading cpu metrics for " + server},
		{Title: "memory", Description: "rss and limits", Action: "loading memory metrics for " + server},
		{Title: "latency", Description: "p50 / p95 / p99", Action: "loading latency metrics for " + server},
	}
}

func envOptions(server string) []MenuItem {
	return []MenuItem{
		{Title: "view all", Description: "list env vars", Action: "listing env vars on " + server},
		{Title: "edit", Description: "change a variable", Action: "opening env editor…"},
		{Title: "import .env", Description: "upload from file", Action: "select .env file to import…"},
	}
}

func networkOptions(server string) []MenuItem {
	return []MenuItem{
		{Title: "ports", Description: "exposed services", Action: "listing ports on " + server},
		{Title: "ingress", Description: "routes and tls", Children: []MenuItem{
			{Title: "routes", Description: "http paths", Action: "listing ingress routes…"},
			{Title: "certificates", Description: "tls certs", Action: "listing certificates…"},
		}},
		{Title: "firewall", Description: "allowed cidrs", Action: "opening firewall rules…"},
	}
}

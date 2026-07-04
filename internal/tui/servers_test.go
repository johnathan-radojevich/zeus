package tui

import (
	"testing"
	"time"

	"github.com/radojevich/zeus/internal/services"
)

func TestServerCacheFresh(t *testing.T) {
	m := Model{
		servers:          ServerMenu(),
		serversFetchedAt: time.Now(),
	}
	if m.serversStale() {
		t.Fatal("expected cache to be fresh immediately after fetch")
	}

	m.serversFetchedAt = time.Now().Add(-serverCacheTTL - time.Second)
	if !m.serversStale() {
		t.Fatal("expected cache to be stale after ttl")
	}
}

func TestServersToMenuItems(t *testing.T) {
	items := serversToMenuItems(services.DefaultServers())
	if len(items) != services.DemoServerCount {
		t.Fatalf("expected %d menu items, got %d", services.DemoServerCount, len(items))
	}
	if items[0].Title != "prod-us-east-1" {
		t.Fatalf("unexpected first item: %q", items[0].Title)
	}
	if !items[0].HasChildren() {
		t.Fatal("expected server item to have action children")
	}
}

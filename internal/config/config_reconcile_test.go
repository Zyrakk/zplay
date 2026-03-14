package config

import (
	"testing"

	"github.com/Zyrakk/zplay/internal/k8s"
)

func TestReconcile_AllMatch(t *testing.T) {
	state := &ServerState{
		Servers: []ServerInfo{
			{Name: "server1", Game: "terraria"},
		},
	}
	discovered := []k8s.DiscoveredServer{
		{Name: "server1", Game: "terraria", Namespace: "zplay-server1"},
	}

	added, orphaned := Reconcile(state, discovered)
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	if len(orphaned) != 0 {
		t.Errorf("expected no orphaned, got %v", orphaned)
	}
}

func TestReconcile_NewInCluster(t *testing.T) {
	state := &ServerState{Servers: []ServerInfo{}}
	discovered := []k8s.DiscoveredServer{
		{Name: "newserver", Game: "terraria", Namespace: "zplay-newserver"},
	}

	added, orphaned := Reconcile(state, discovered)
	if len(added) != 1 || added[0] != "newserver" {
		t.Errorf("expected added=[newserver], got %v", added)
	}
	if len(orphaned) != 0 {
		t.Errorf("expected no orphaned, got %v", orphaned)
	}
}

func TestReconcile_OrphanedLocal(t *testing.T) {
	state := &ServerState{
		Servers: []ServerInfo{
			{Name: "gone", Game: "terraria"},
		},
	}
	discovered := []k8s.DiscoveredServer{}

	added, orphaned := Reconcile(state, discovered)
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	if len(orphaned) != 1 || orphaned[0] != "gone" {
		t.Errorf("expected orphaned=[gone], got %v", orphaned)
	}
}

func TestReconcile_MixedState(t *testing.T) {
	state := &ServerState{
		Servers: []ServerInfo{
			{Name: "kept", Game: "terraria"},
			{Name: "removed", Game: "terraria"},
		},
	}
	discovered := []k8s.DiscoveredServer{
		{Name: "kept", Game: "terraria", Namespace: "zplay-kept"},
		{Name: "newone", Game: "terraria", Namespace: "zplay-newone"},
	}

	added, orphaned := Reconcile(state, discovered)
	if len(added) != 1 || added[0] != "newone" {
		t.Errorf("expected added=[newone], got %v", added)
	}
	if len(orphaned) != 1 || orphaned[0] != "removed" {
		t.Errorf("expected orphaned=[removed], got %v", orphaned)
	}
}

func TestServerState_AddRemoveGet(t *testing.T) {
	state := &ServerState{}

	state.Add(ServerInfo{Name: "test1", Game: "terraria", Port: 7777})
	if len(state.Servers) != 1 {
		t.Fatal("expected 1 server after add")
	}

	got := state.Get("test1")
	if got == nil || got.Name != "test1" {
		t.Error("Get should return added server")
	}

	if state.Get("nonexistent") != nil {
		t.Error("Get should return nil for nonexistent")
	}

	if !state.Remove("test1") {
		t.Error("Remove should return true for existing")
	}
	if len(state.Servers) != 0 {
		t.Error("expected 0 servers after remove")
	}

	if state.Remove("nonexistent") {
		t.Error("Remove should return false for nonexistent")
	}
}

func TestServerState_NextPort(t *testing.T) {
	state := &ServerState{
		Servers: []ServerInfo{
			{Name: "s1", Game: "terraria", Port: 7777},
			{Name: "s2", Game: "terraria", Port: 7778},
		},
	}

	next := state.NextPort("terraria", 7777)
	if next != 7779 {
		t.Errorf("expected next port 7779, got %d", next)
	}

	next = state.NextPort("minecraft", 25565)
	if next != 25565 {
		t.Errorf("expected next port 25565 (no minecraft servers), got %d", next)
	}
}

func TestDefaultConfig_HasSensibleDefaults(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Backup.Path != "/mnt/das/zplay-backups" {
		t.Errorf("unexpected backup path: %s", cfg.Backup.Path)
	}
	if cfg.Backup.Retention != 7 {
		t.Errorf("unexpected retention: %d", cfg.Backup.Retention)
	}
	if cfg.Storage.Size != "10Gi" {
		t.Errorf("unexpected storage size: %s", cfg.Storage.Size)
	}
	if cfg.Defaults.MemoryRequest != "4Gi" {
		t.Errorf("unexpected memory request: %s", cfg.Defaults.MemoryRequest)
	}
	if cfg.Probes.VanillaInitialDelay != 120 {
		t.Errorf("unexpected vanilla probe delay: %d", cfg.Probes.VanillaInitialDelay)
	}
}

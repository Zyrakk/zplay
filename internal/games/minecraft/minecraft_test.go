package minecraft

import (
	"strings"
	"testing"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
)

func testConfig() *games.ServerConfig {
	cfg := games.NewServerConfig(&config.Config{
		Domain: "test.example.com",
		Backup: config.BackupConfig{
			Path: "/mnt/test-backups", Schedule: "0 3 * * *", Retention: 5, Node: "testnode",
		},
		Storage:  config.StorageConfig{Size: "20Gi", Class: "test-storage"},
		Defaults: config.DefaultsConfig{MemoryRequest: "2Gi", MemoryLimit: "4Gi", CPURequest: "250m", CPULimit: "1"},
		Probes:   config.ProbesConfig{VanillaInitialDelay: 90, TmodloaderInitialDelay: 240},
	})
	cfg.Name = "mctest"
	cfg.Game = "minecraft"
	cfg.Port = 25565
	cfg.MaxPlayers = 10
	return cfg
}

func TestRenderManifests_Vanilla(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "vanilla"

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	checks := []struct {
		desc     string
		contains string
	}{
		{"namespace", "namespace: zplay-mctest"},
		{"storage class", "storageClassName: test-storage"},
		{"storage size", "storage: 20Gi"},
		{"minecraft image", "image: itzg/minecraft-server:latest"},
		{"EULA", `value: "TRUE"`},
		{"TYPE vanilla", `value: "VANILLA"`},
		{"memory request", "memory: 2Gi"},
		{"cpu request", `cpu: "250m"`},
		{"RCON enabled", `value: "true"`},
		{"mc-health probe", "mc-health"},
		{"game port", "containerPort: 25565"},
		{"rcon port", "containerPort: 25575"},
	}

	for _, c := range checks {
		if !strings.Contains(joined, c.contains) {
			t.Errorf("expected manifests to contain %q (%s)", c.contains, c.desc)
		}
	}
}

func TestRenderManifests_Paper(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "paper"

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, `value: "PAPER"`) {
		t.Error("paper variant should set TYPE=PAPER")
	}
}

func TestRenderManifests_Forge(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "forge"

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, `value: "FORGE"`) {
		t.Error("forge variant should set TYPE=FORGE")
	}
}

func TestRenderManifests_WithVersion(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	cfg.Version = "1.20.4"

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, `value: "1.20.4"`) {
		t.Error("version should appear in manifests")
	}
}

func TestRenderManifests_WithOpsAndMOTD(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	cfg.Ops = "player1,player2"
	cfg.MOTD = "Welcome to my server"

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, "player1,player2") {
		t.Error("ops should appear in manifests")
	}
	if !strings.Contains(joined, "Welcome to my server") {
		t.Error("MOTD should appear in manifests")
	}
}

func TestRenderManifests_WithAutoBackup(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	cfg.AutoBackup = true

	manifests, err := mc.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, "CronJob") {
		t.Error("auto-backup should include CronJob")
	}
	if !strings.Contains(joined, "/mnt/test-backups") {
		t.Error("cronjob should use configured backup path")
	}
}

func TestRenderBackupJob(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Timestamp = "20260316-120000"

	manifest, err := mc.RenderBackupJob(cfg)
	if err != nil {
		t.Fatalf("RenderBackupJob failed: %v", err)
	}

	if !strings.Contains(manifest, "/mnt/test-backups") {
		t.Error("backup job should use configured backup path")
	}
	if !strings.Contains(manifest, "testnode") {
		t.Error("backup job should use configured backup node")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	if err := mc.Validate(cfg); err != nil {
		t.Errorf("expected valid config, got: %v", err)
	}
}

func TestValidate_InvalidVariant(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Variant = "bukkit"
	if err := mc.Validate(cfg); err == nil {
		t.Error("expected error for invalid variant")
	}
}

func TestValidate_EmptyName(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.Name = ""
	if err := mc.Validate(cfg); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidate_InvalidMaxPlayers(t *testing.T) {
	mc := &Minecraft{}
	cfg := testConfig()
	cfg.MaxPlayers = 0
	if err := mc.Validate(cfg); err == nil {
		t.Error("expected error for 0 max players")
	}
}

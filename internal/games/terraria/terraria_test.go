package terraria

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
	cfg.Name = "testserver"
	cfg.Game = "terraria"
	cfg.Port = 7777
	cfg.MaxPlayers = 4
	cfg.WorldSize = "small"
	cfg.Difficulty = "1"
	return cfg
}

func TestRenderManifests_VanillaNoPassword(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "vanilla"

	manifests, err := terraria.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	checks := []struct {
		desc     string
		contains string
	}{
		{"namespace", "namespace: zplay-testserver"},
		{"storage class", "storageClassName: test-storage"},
		{"storage size", "storage: 20Gi"},
		{"vanilla image", "image: hexlo/terraria-server-docker:latest"},
		{"memory request", "memory: 2Gi"},
		{"memory limit", "memory: 4Gi"},
		{"cpu request", `cpu: "250m"`},
		{"cpu limit", `cpu: "1"`},
		{"world env", "world"},
	}

	for _, c := range checks {
		if c.contains != "" && !strings.Contains(joined, c.contains) {
			t.Errorf("expected manifests to contain %q (%s)", c.contains, c.desc)
		}
	}

	if strings.Contains(joined, "TMOD_WORLDNAME") {
		t.Error("vanilla manifests should not contain TMOD_ vars")
	}
}

func TestRenderManifests_TmodloaderNoPassword(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "tmodloader"

	manifests, err := terraria.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, "jacobsmile/tmodloader1.4:latest") {
		t.Error("tmodloader manifests should use jacobsmile image")
	}
	if !strings.Contains(joined, "TMOD_PASS") {
		t.Error("tmodloader manifests should always include TMOD_PASS")
	}
	if !strings.Contains(joined, `value: ""`) {
		t.Error("tmodloader without password should set TMOD_PASS to empty string")
	}
}

func TestRenderManifests_WithPassword(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "tmodloader"
	cfg.Password = "secret123"

	manifests, err := terraria.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, "secretKeyRef") {
		t.Error("manifests with password should reference secret")
	}
}

func TestRenderManifests_WithAutoBackup(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	cfg.AutoBackup = true

	manifests, err := terraria.RenderManifests(cfg)
	if err != nil {
		t.Fatalf("RenderManifests failed: %v", err)
	}

	joined := strings.Join(manifests, "\n---\n")

	if !strings.Contains(joined, "CronJob") {
		t.Error("auto-backup manifests should include CronJob")
	}
	if !strings.Contains(joined, `schedule: "0 3 * * *"`) {
		t.Error("cronjob should use configured schedule")
	}
	if !strings.Contains(joined, "testnode") {
		t.Error("cronjob should use configured backup node")
	}
	if !strings.Contains(joined, "/mnt/test-backups") {
		t.Error("cronjob should use configured backup path")
	}
}

func TestRenderBackupJob(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Timestamp = "20260314-120000"

	manifest, err := terraria.RenderBackupJob(cfg)
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
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "vanilla"
	if err := terraria.Validate(cfg); err != nil {
		t.Errorf("expected valid config, got: %v", err)
	}
}

func TestValidate_InvalidName(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Name = ""
	if err := terraria.Validate(cfg); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidate_InvalidVariant(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.Variant = "invalid"
	if err := terraria.Validate(cfg); err == nil {
		t.Error("expected error for invalid variant")
	}
}

func TestValidate_InvalidWorldSize(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.WorldSize = "huge"
	if err := terraria.Validate(cfg); err == nil {
		t.Error("expected error for invalid world size")
	}
}

func TestValidate_InvalidMaxPlayers(t *testing.T) {
	terraria := &Terraria{}
	cfg := testConfig()
	cfg.MaxPlayers = 0
	if err := terraria.Validate(cfg); err == nil {
		t.Error("expected error for 0 max players")
	}
}

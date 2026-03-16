package minecraft

import (
	"embed"
	"fmt"

	"github.com/Zyrakk/zplay/internal/games"
)

//go:embed templates/*.yaml
var templates embed.FS

func init() {
	games.Register(&Minecraft{})
}

type Minecraft struct{}

func (m *Minecraft) Name() string        { return "minecraft" }
func (m *Minecraft) DisplayName() string { return "Minecraft" }
func (m *Minecraft) DefaultPort() int    { return 25565 }

func (m *Minecraft) Validate(cfg *games.ServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if len(cfg.Name) > 20 {
		return fmt.Errorf("server name too long (max 20 characters)")
	}
	if cfg.MaxPlayers < 1 || cfg.MaxPlayers > 100 {
		return fmt.Errorf("max players must be between 1 and 100")
	}

	if cfg.Variant == "" {
		cfg.Variant = "vanilla"
	}
	validVariants := map[string]bool{"vanilla": true, "paper": true, "forge": true}
	if !validVariants[cfg.Variant] {
		return fmt.Errorf("variant must be vanilla, paper, or forge")
	}

	// Map variant to itzg/minecraft-server TYPE env var
	if cfg.ServerType == "" {
		typeMap := map[string]string{
			"vanilla": "VANILLA",
			"paper":   "PAPER",
			"forge":   "FORGE",
		}
		cfg.ServerType = typeMap[cfg.Variant]
	}

	return nil
}

func (m *Minecraft) RenderManifests(cfg *games.ServerConfig) ([]string, error) {
	cfg.Game = m.Name()
	cfg.Entrypoint = games.PortToEntrypoint(cfg.Game, cfg.Port)

	if cfg.Variant == "" {
		cfg.Variant = "vanilla"
	}
	// Ensure ServerType is set (may not have gone through Validate)
	if cfg.ServerType == "" {
		typeMap := map[string]string{
			"vanilla": "VANILLA",
			"paper":   "PAPER",
			"forge":   "FORGE",
		}
		cfg.ServerType = typeMap[cfg.Variant]
	}

	// Set probe delays — Minecraft typically starts faster than tModLoader
	if cfg.ProbeInitialDelay == 0 {
		cfg.ProbeInitialDelay = 120
	}
	if cfg.ProbeReadinessDelay == 0 {
		cfg.ProbeReadinessDelay = 60
	}

	// Infrastructure defaults
	games.ApplyInfraDefaults(cfg)

	templateFiles := []string{
		"namespace.yaml",
		"volume.yaml",
		"secret.yaml",
		"deployment.yaml",
		"service.yaml",
		"ingress.yaml",
	}
	if cfg.AutoBackup {
		templateFiles = append(templateFiles, "cronjob-backup.yaml")
	}

	return games.RenderTemplates(templates, "templates", templateFiles, cfg)
}

func (m *Minecraft) RenderBackupJob(cfg *games.ServerConfig) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("server name is required")
	}
	if cfg.Timestamp == "" {
		return "", fmt.Errorf("backup timestamp is required")
	}

	cfg.Game = m.Name()
	games.ApplyInfraDefaults(cfg)

	return games.RenderSingleTemplate(templates, "templates", "backup-job.yaml", cfg)
}

func (m *Minecraft) RenderRestoreJob(cfg *games.ServerConfig) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("server name is required")
	}
	if cfg.Timestamp == "" {
		return "", fmt.Errorf("restore timestamp is required")
	}
	if cfg.BackupFile == "" {
		return "", fmt.Errorf("backup file is required")
	}

	cfg.Game = m.Name()
	games.ApplyInfraDefaults(cfg)

	return games.RenderSingleTemplate(templates, "templates", "restore-job.yaml", cfg)
}

func (m *Minecraft) GetDeploymentName(serverName string) string {
	return serverName + "-minecraft"
}

func (m *Minecraft) GetNamespace(serverName string) string {
	return "zplay-" + serverName
}

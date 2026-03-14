package terraria

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/util"
)

//go:embed templates/*.yaml
var templates embed.FS

func init() {
	games.Register(&Terraria{})
}

type Terraria struct{}

func (t *Terraria) Name() string        { return "terraria" }
func (t *Terraria) DisplayName() string { return "Terraria" }
func (t *Terraria) DefaultPort() int    { return 7777 }

func (t *Terraria) Validate(cfg *games.ServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if len(cfg.Name) > 20 {
		return fmt.Errorf("server name too long (max 20 characters)")
	}

	validSizes := map[string]bool{"small": true, "medium": true, "large": true}
	if cfg.WorldSize != "" && !validSizes[cfg.WorldSize] {
		return fmt.Errorf("world size must be small, medium, or large")
	}

	if cfg.MaxPlayers < 1 || cfg.MaxPlayers > 255 {
		return fmt.Errorf("max players must be between 1 and 255")
	}

	if cfg.Variant == "" {
		cfg.Variant = "vanilla"
	}
	if cfg.Variant != "vanilla" && cfg.Variant != "tmodloader" {
		return fmt.Errorf("variant must be vanilla or tmodloader")
	}
	if cfg.Variant == "tmodloader" && cfg.NodeSelector != "lake" {
		return fmt.Errorf("tModLoader requires an x86 node. Only lake is available.")
	}
	if cfg.Variant == "tmodloader" {
		if memoryMi, err := util.MemoryToMi(cfg.Memory); err == nil && memoryMi < 4*1024 {
			fmt.Println("⚠ tModLoader with mods like Calamity recommends at least 4Gi of memory")
		}
	}

	validDifficulties := map[string]bool{
		"0": true,
		"1": true,
		"2": true,
		"3": true,
	}
	if cfg.Difficulty != "" && !validDifficulties[cfg.Difficulty] {
		return fmt.Errorf("difficulty must be 0, 1, 2, or 3")
	}

	return nil
}

func (t *Terraria) RenderManifests(cfg *games.ServerConfig) ([]string, error) {
	// Convert world size to numeric value
	worldSizeMap := map[string]int{
		"small":  1,
		"medium": 2,
		"large":  3,
	}
	if cfg.WorldSize == "" {
		cfg.WorldSize = "medium"
	}
	cfg.WorldSizeNum = worldSizeMap[cfg.WorldSize]
	if cfg.Difficulty == "" {
		cfg.Difficulty = "0"
	}
	if cfg.Variant == "" {
		cfg.Variant = "vanilla"
	}

	// Set game name
	cfg.Game = t.Name()
	cfg.Entrypoint = games.PortToEntrypoint(cfg.Game, cfg.Port)

	// Set probe delays based on variant, using config values as defaults
	if cfg.Variant == "tmodloader" {
		if cfg.ProbeInitialDelay == 0 {
			if cfg.ProbeTmodloaderInitDelay > 0 {
				cfg.ProbeInitialDelay = cfg.ProbeTmodloaderInitDelay
			} else {
				cfg.ProbeInitialDelay = 300
			}
		}
		if cfg.ProbeReadinessDelay == 0 {
			cfg.ProbeReadinessDelay = cfg.ProbeInitialDelay * 3 / 5
		}
	} else {
		if cfg.ProbeInitialDelay == 0 {
			if cfg.ProbeVanillaInitDelay > 0 {
				cfg.ProbeInitialDelay = cfg.ProbeVanillaInitDelay
			} else {
				cfg.ProbeInitialDelay = 120
			}
		}
		if cfg.ProbeReadinessDelay == 0 {
			cfg.ProbeReadinessDelay = cfg.ProbeInitialDelay / 2
		}
	}

	// Set infrastructure defaults
	if cfg.StorageSize == "" {
		cfg.StorageSize = "10Gi"
	}
	if cfg.StorageClass == "" {
		cfg.StorageClass = "nfs-shared"
	}
	if cfg.CPURequest == "" {
		cfg.CPURequest = "500m"
	}
	if cfg.CPULimit == "" {
		cfg.CPULimit = "2"
	}
	if cfg.BackupPath == "" {
		cfg.BackupPath = "/mnt/das/zplay-backups"
	}
	if cfg.BackupNode == "" {
		cfg.BackupNode = "oracle1"
	}
	if cfg.BackupSchedule == "" {
		cfg.BackupSchedule = "0 4 * * *"
	}
	if cfg.BackupRetention == 0 {
		cfg.BackupRetention = 7
	}

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

	var manifests []string

	for _, file := range templateFiles {
		tmpl, err := template.ParseFS(templates, "templates/"+file)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", file, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cfg); err != nil {
			return nil, fmt.Errorf("executing template %s: %w", file, err)
		}

		manifests = append(manifests, buf.String())
	}

	// Filter empty manifests (conditional templates like secret.yaml)
	var filtered []string
	for _, m := range manifests {
		if strings.TrimSpace(m) != "" {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

func (t *Terraria) RenderBackupJob(cfg *games.ServerConfig) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("server name is required")
	}
	if cfg.Timestamp == "" {
		return "", fmt.Errorf("backup timestamp is required")
	}

	cfg.Game = t.Name()
	if cfg.BackupPath == "" {
		cfg.BackupPath = "/mnt/das/zplay-backups"
	}
	if cfg.BackupNode == "" {
		cfg.BackupNode = "oracle1"
	}

	tmpl, err := template.ParseFS(templates, "templates/backup-job.yaml")
	if err != nil {
		return "", fmt.Errorf("parsing template backup-job.yaml: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("executing template backup-job.yaml: %w", err)
	}

	return buf.String(), nil
}

func (t *Terraria) RenderRestoreJob(cfg *games.ServerConfig) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("server name is required")
	}
	if cfg.Timestamp == "" {
		return "", fmt.Errorf("restore timestamp is required")
	}
	if cfg.BackupFile == "" {
		return "", fmt.Errorf("backup file is required")
	}

	cfg.Game = t.Name()
	if cfg.BackupPath == "" {
		cfg.BackupPath = "/mnt/das/zplay-backups"
	}
	if cfg.BackupNode == "" {
		cfg.BackupNode = "oracle1"
	}

	tmpl, err := template.ParseFS(templates, "templates/restore-job.yaml")
	if err != nil {
		return "", fmt.Errorf("parsing template restore-job.yaml: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("executing template restore-job.yaml: %w", err)
	}

	return buf.String(), nil
}

func (t *Terraria) GetDeploymentName(serverName string) string {
	return serverName + "-terraria"
}

func (t *Terraria) GetNamespace(serverName string) string {
	return "zplay-" + serverName
}

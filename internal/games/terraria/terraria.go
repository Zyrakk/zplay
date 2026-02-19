package terraria

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Zyrakk/zplay/internal/games"
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
		if memoryMi, err := memoryToMi(cfg.Memory); err == nil && memoryMi < 4*1024 {
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

	templateFiles := []string{
		"namespace.yaml",
		"volume.yaml",
		"secret.yaml",
		"deployment.yaml",
		"service.yaml",
		"ingress.yaml",
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

func (t *Terraria) GetDeploymentName(serverName string) string {
	return serverName + "-terraria"
}

func (t *Terraria) GetNamespace(serverName string) string {
	return "zplay-" + serverName
}

func memoryToMi(memory string) (int, error) {
	if len(memory) < 3 {
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}

	value, err := strconv.Atoi(memory[:len(memory)-2])
	if err != nil {
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}

	switch memory[len(memory)-2] {
	case 'G':
		return value * 1024, nil
	case 'M':
		return value, nil
	default:
		return 0, fmt.Errorf("invalid memory format: %s", memory)
	}
}

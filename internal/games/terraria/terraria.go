package terraria

import (
	"bytes"
	"embed"
	"fmt"
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

	// Set game name
	cfg.Game = t.Name()

	templateFiles := []string{
		"namespace.yaml",
		"volume.yaml",
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

	return manifests, nil
}

func (t *Terraria) GetDeploymentName(serverName string) string {
	return serverName + "-terraria"
}

func (t *Terraria) GetNamespace(serverName string) string {
	return "zplay-" + serverName
}

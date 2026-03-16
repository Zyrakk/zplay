package games

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

// ApplyInfraDefaults sets fallback values for infrastructure config fields
// that should have been set by NewServerConfig but may be empty in manually
// constructed ServerConfigs (backup/restore flows).
func ApplyInfraDefaults(cfg *ServerConfig) {
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
}

// RenderTemplates parses and executes a list of template files from an embedded FS,
// filtering out empty results (from conditional templates like secret.yaml).
func RenderTemplates(fs embed.FS, dir string, files []string, cfg *ServerConfig) ([]string, error) {
	var manifests []string

	for _, file := range files {
		tmpl, err := template.ParseFS(fs, dir+"/"+file)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", file, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cfg); err != nil {
			return nil, fmt.Errorf("executing template %s: %w", file, err)
		}

		manifests = append(manifests, buf.String())
	}

	var filtered []string
	for _, m := range manifests {
		if strings.TrimSpace(m) != "" {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// RenderSingleTemplate parses and executes a single template file.
func RenderSingleTemplate(fs embed.FS, dir string, file string, cfg *ServerConfig) (string, error) {
	tmpl, err := template.ParseFS(fs, dir+"/"+file)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", file, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("executing template %s: %w", file, err)
	}

	return buf.String(), nil
}

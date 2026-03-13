package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zyrakk/zplay/internal/k8s"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Domain       string `yaml:"domain"`
	Kubeconfig   string `yaml:"kubeconfig"`
	NodeSelector string `yaml:"node_selector"`
	DataPath     string `yaml:"data_path"`
}

type ServerState struct {
	Servers []ServerInfo `yaml:"servers"`
}

type ServerInfo struct {
	Name       string `yaml:"name"`
	Game       string `yaml:"game"`
	Namespace  string `yaml:"namespace,omitempty"`
	Variant    string `yaml:"variant,omitempty"`
	AutoBackup bool   `yaml:"auto_backup,omitempty"`
	Node       string `yaml:"node"`
	Port       int    `yaml:"port"`
	Memory     string `yaml:"memory"`
	MaxPlayers int    `yaml:"max_players"`
	CreatedAt  string `yaml:"created_at"`
}

func defaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Domain:       "play.zyrak.cloud",
		Kubeconfig:   resolveKubeconfig("", home),
		NodeSelector: "",
		DataPath:     filepath.Join(home, ".zplay"),
	}
}

func Load() (*Config, error) {
	cfg := defaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configPath := filepath.Join(home, ".zplay", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist, create default
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				return nil, err
			}
			return cfg, Save(cfg)
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Kubeconfig = resolveKubeconfig(cfg.Kubeconfig, home)

	return cfg, nil
}

func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".zplay", "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func LoadServerState(cfg *Config) (*ServerState, error) {
	statePath := filepath.Join(cfg.DataPath, "servers.yaml")
	state := &ServerState{Servers: []ServerInfo{}}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, err
	}

	return state, nil
}

func SaveServerState(cfg *Config, state *ServerState) error {
	statePath := filepath.Join(cfg.DataPath, "servers.yaml")
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}

func (s *ServerState) Add(server ServerInfo) {
	s.Servers = append(s.Servers, server)
}

func (s *ServerState) Remove(name string) bool {
	for i, srv := range s.Servers {
		if srv.Name == name {
			s.Servers = append(s.Servers[:i], s.Servers[i+1:]...)
			return true
		}
	}
	return false
}

func Reconcile(state *ServerState, discovered []k8s.DiscoveredServer) (added []string, orphaned []string) {
	localByName := make(map[string]struct{}, len(state.Servers))
	addedByName := make(map[string]struct{})
	orphanedByName := make(map[string]struct{})
	for _, srv := range state.Servers {
		if srv.Name == "" {
			continue
		}
		localByName[srv.Name] = struct{}{}
	}

	discoveredByName := make(map[string]struct{}, len(discovered))
	for _, srv := range discovered {
		if srv.Name == "" {
			continue
		}
		discoveredByName[srv.Name] = struct{}{}
		if _, exists := localByName[srv.Name]; !exists {
			if _, alreadyAdded := addedByName[srv.Name]; alreadyAdded {
				continue
			}
			added = append(added, srv.Name)
			addedByName[srv.Name] = struct{}{}
		}
	}

	for _, srv := range state.Servers {
		if srv.Name == "" {
			continue
		}
		if _, exists := discoveredByName[srv.Name]; !exists {
			if _, alreadyOrphaned := orphanedByName[srv.Name]; alreadyOrphaned {
				continue
			}
			orphaned = append(orphaned, srv.Name)
			orphanedByName[srv.Name] = struct{}{}
		}
	}

	sort.Strings(added)
	sort.Strings(orphaned)

	return added, orphaned
}

func (s *ServerState) Get(name string) *ServerInfo {
	for _, srv := range s.Servers {
		if srv.Name == name {
			return &srv
		}
	}
	return nil
}

func (s *ServerState) NextPort(game string, basePort int) int {
	maxPort := basePort - 1
	for _, srv := range s.Servers {
		if srv.Game == game && srv.Port > maxPort {
			maxPort = srv.Port
		}
	}
	return maxPort + 1
}

func resolveKubeconfig(configuredValue, home string) string {
	legacyDefault := filepath.Join(home, ".zcloud", "kubeconfig")
	configured := normalizeKubeconfigValue(configuredValue, home)

	// Keep explicit non-empty user configuration even if the file doesn't exist.
	if configured != "" && !isLegacyDefaultKubeconfig(configured, home) {
		return configured
	}

	// Keep legacy default only when present; otherwise continue fallback chain.
	if configured != "" && fileExistsAny(configured) {
		return configured
	}

	if envKubeconfig := normalizeKubeconfigValue(os.Getenv("KUBECONFIG"), home); envKubeconfig != "" {
		return envKubeconfig
	}

	homeKubeconfig := filepath.Join(home, ".kube", "config")
	if fileExistsAny(homeKubeconfig) {
		return homeKubeconfig
	}

	if fileExistsAny(legacyDefault) {
		return legacyDefault
	}

	// Preserve backward-compatible final fallback.
	return legacyDefault
}

func normalizeKubeconfigValue(value, home string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	parts := strings.Split(value, string(os.PathListSeparator))
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		path := expandPath(part, home)
		if path == "" {
			continue
		}
		normalized = append(normalized, path)
	}

	if len(normalized) == 0 {
		return ""
	}
	return strings.Join(normalized, string(os.PathListSeparator))
}

func expandPath(value, home string) string {
	path := strings.TrimSpace(value)
	if path == "" {
		return ""
	}

	if path == "~" {
		path = home
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return os.ExpandEnv(path)
}

func fileExistsAny(value string) bool {
	parts := strings.Split(value, string(os.PathListSeparator))
	for _, part := range parts {
		path := strings.TrimSpace(part)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func isLegacyDefaultKubeconfig(value, home string) bool {
	parts := strings.Split(value, string(os.PathListSeparator))
	if len(parts) != 1 {
		return false
	}

	legacyDefault := filepath.Clean(filepath.Join(home, ".zcloud", "kubeconfig"))
	return filepath.Clean(strings.TrimSpace(parts[0])) == legacyDefault
}

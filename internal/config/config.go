package config

import (
	"os"
	"path/filepath"

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
		Kubeconfig:   filepath.Join(home, ".zcloud", "kubeconfig"),
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

	return os.WriteFile(configPath, data, 0644)
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

	return os.WriteFile(statePath, data, 0644)
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

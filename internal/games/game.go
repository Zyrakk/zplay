package games

import (
	"fmt"

	"github.com/Zyrakk/zplay/internal/config"
)

// ServerConfig holds the configuration for deploying a game server
type ServerConfig struct {
	Name         string
	Timestamp    string
	BackupFile   string
	Game         string
	Variant      string
	AutoBackup   bool
	Memory       string
	MemoryLimit  string
	Port         int
	Entrypoint   string
	Password     string
	MOTD         string
	MaxPlayers   int
	NodeSelector string
	Domain       string

	// Infrastructure config (from config.yaml)
	BackupPath               string
	BackupSchedule           string
	BackupRetention          int
	BackupNode               string
	StorageSize              string
	StorageClass             string
	CPURequest               string
	CPULimit                 string
	ProbeInitialDelay        int
	ProbeReadinessDelay      int
	ProbeVanillaInitDelay    int
	ProbeTmodloaderInitDelay int

	// Terraria specific
	WorldSize    string
	WorldSizeNum int
	Difficulty   string

	// Minecraft specific (future)
	ServerType string // PAPER, FABRIC, FORGE, etc.
	Version    string
	Mods       []string
}

// Game interface defines what each game implementation must provide
type Game interface {
	// Name returns the game identifier (e.g., "terraria", "minecraft")
	Name() string

	// DisplayName returns the human-readable name
	DisplayName() string

	// DefaultPort returns the default port for this game
	DefaultPort() int

	// Validate checks if the server config is valid
	Validate(cfg *ServerConfig) error

	// RenderManifests generates Kubernetes manifests for the server
	RenderManifests(cfg *ServerConfig) ([]string, error)

	// RenderBackupJob generates a Kubernetes Job manifest for manual backup
	RenderBackupJob(cfg *ServerConfig) (string, error)

	// RenderRestoreJob generates a Kubernetes Job manifest for restore
	RenderRestoreJob(cfg *ServerConfig) (string, error)

	// GetDeploymentName returns the deployment name for a server
	GetDeploymentName(serverName string) string

	// GetNamespace returns the namespace for a server
	GetNamespace(serverName string) string
}

// Registry holds available games
var registry = make(map[string]Game)

// Register adds a game to the registry
func Register(game Game) {
	registry[game.Name()] = game
}

// Get returns a game by name
func Get(name string) Game {
	return registry[name]
}

// Available returns all registered games
func Available() []Game {
	games := make([]Game, 0, len(registry))
	for _, g := range registry {
		games = append(games, g)
	}
	return games
}

// AvailableNames returns names of all registered games
func AvailableNames() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// NewServerConfig creates a ServerConfig with defaults
func NewServerConfig(cfg *config.Config) *ServerConfig {
	return &ServerConfig{
		Variant:                  "vanilla",
		AutoBackup:               true,
		Memory:                   cfg.Defaults.MemoryRequest,
		MemoryLimit:              cfg.Defaults.MemoryLimit,
		Difficulty:               "0",
		MaxPlayers:               8,
		NodeSelector:             cfg.NodeSelector,
		Domain:                   cfg.Domain,
		BackupPath:               cfg.Backup.Path,
		BackupSchedule:           cfg.Backup.Schedule,
		BackupRetention:          cfg.Backup.Retention,
		BackupNode:               cfg.Backup.Node,
		StorageSize:              cfg.Storage.Size,
		StorageClass:             cfg.Storage.Class,
		CPURequest:               cfg.Defaults.CPURequest,
		CPULimit:                 cfg.Defaults.CPULimit,
		ProbeVanillaInitDelay:    cfg.Probes.VanillaInitialDelay,
		ProbeTmodloaderInitDelay: cfg.Probes.TmodloaderInitialDelay,
	}
}

// PortToEntrypoint maps known game ports to fixed Traefik entrypoints.
func PortToEntrypoint(game string, port int) string {
	entrypoints := map[string]map[int]string{
		"terraria": {
			7777: "terraria1",
			7778: "terraria2",
		},
		"minecraft": {
			25565: "minecraft1",
			25566: "minecraft2",
		},
	}

	if gameMap, ok := entrypoints[game]; ok {
		if ep, ok := gameMap[port]; ok {
			return ep
		}
	}

	return fmt.Sprintf("zplay-%d", port)
}

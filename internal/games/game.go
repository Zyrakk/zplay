package games

import "github.com/Zyrakk/zplay/internal/config"

// ServerConfig holds the configuration for deploying a game server
type ServerConfig struct {
	Name         string
	Game         string
	Memory       string
	MemoryLimit  string
	Port         int
	Password     string
	MaxPlayers   int
	NodeSelector string
	Domain       string

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
		Memory:       "4Gi",
		MemoryLimit:  "8Gi",
		MaxPlayers:   8,
		NodeSelector: cfg.NodeSelector,
		Domain:       cfg.Domain,
	}
}

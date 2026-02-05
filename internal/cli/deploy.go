package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunDeploy(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Deploy New Server"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Select game
	availableGames := games.Available()
	if len(availableGames) == 0 {
		return fmt.Errorf("no games available")
	}

	fmt.Println("Select game:")
	for i, g := range availableGames {
		fmt.Printf("  %d) %s\n", i+1, g.DisplayName())
	}
	fmt.Print("\nChoice [1]: ")
	gameChoice, _ := reader.ReadString('\n')
	gameChoice = strings.TrimSpace(gameChoice)

	gameIdx := 0
	if gameChoice != "" {
		idx, err := strconv.Atoi(gameChoice)
		if err != nil || idx < 1 || idx > len(availableGames) {
			return fmt.Errorf("invalid game selection")
		}
		gameIdx = idx - 1
	}

	game := availableGames[gameIdx]
	serverCfg := games.NewServerConfig(cfg)
	serverCfg.Game = game.Name()

	// Load server state to get next available port
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	// Server name
	fmt.Print("\nServer name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name is required")
	}
	serverCfg.Name = name

	// Check if server already exists
	if state.Get(name) != nil {
		return fmt.Errorf("server '%s' already exists", name)
	}

	// Memory
	fmt.Printf("Memory [%s]: ", serverCfg.Memory)
	memory, _ := reader.ReadString('\n')
	memory = strings.TrimSpace(memory)
	if memory != "" {
		serverCfg.Memory = memory
	}

	// Memory limit (double the request by default)
	memVal := strings.TrimSuffix(serverCfg.Memory, "Gi")
	if memInt, err := strconv.Atoi(memVal); err == nil {
		serverCfg.MemoryLimit = fmt.Sprintf("%dGi", memInt*2)
	}

	// Game-specific options
	switch game.Name() {
	case "terraria":
		fmt.Println("\nWorld size:")
		fmt.Println("  1) small")
		fmt.Println("  2) medium")
		fmt.Println("  3) large")
		fmt.Print("Choice [2]: ")
		sizeChoice, _ := reader.ReadString('\n')
		sizeChoice = strings.TrimSpace(sizeChoice)

		switch sizeChoice {
		case "1":
			serverCfg.WorldSize = "small"
		case "3":
			serverCfg.WorldSize = "large"
		default:
			serverCfg.WorldSize = "medium"
		}
	}

	// Max players
	fmt.Printf("Max players [%d]: ", serverCfg.MaxPlayers)
	maxPlayers, _ := reader.ReadString('\n')
	maxPlayers = strings.TrimSpace(maxPlayers)
	if maxPlayers != "" {
		mp, err := strconv.Atoi(maxPlayers)
		if err != nil {
			return fmt.Errorf("invalid max players value")
		}
		serverCfg.MaxPlayers = mp
	}

	// Password
	fmt.Print("Password (optional): ")
	password, _ := reader.ReadString('\n')
	serverCfg.Password = strings.TrimSpace(password)

	// Port
	nextPort := state.NextPort(game.Name(), game.DefaultPort())
	fmt.Printf("Port [%d]: ", nextPort)
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port value")
		}
		serverCfg.Port = port
	} else {
		serverCfg.Port = nextPort
	}

	// Validate
	if err := game.Validate(serverCfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Confirm
	fmt.Println()
	fmt.Println(dimStyle.Render("─────────────────────────────"))
	fmt.Printf("Game:        %s\n", game.DisplayName())
	fmt.Printf("Server:      %s\n", serverCfg.Name)
	fmt.Printf("Memory:      %s (limit: %s)\n", serverCfg.Memory, serverCfg.MemoryLimit)
	fmt.Printf("Max Players: %d\n", serverCfg.MaxPlayers)
	fmt.Printf("Port:        %d\n", serverCfg.Port)
	if serverCfg.WorldSize != "" {
		fmt.Printf("World Size:  %s\n", serverCfg.WorldSize)
	}
	if serverCfg.Password != "" {
		fmt.Printf("Password:    %s\n", "********")
	}
	fmt.Println(dimStyle.Render("─────────────────────────────"))

	fmt.Print("\nDeploy? [Y/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Deploy
	fmt.Println()
	printInfo("Generating manifests...")

	manifests, err := game.RenderManifests(serverCfg)
	if err != nil {
		return fmt.Errorf("rendering manifests: %w", err)
	}

	client := k8s.NewClient(cfg)

	printInfo("Applying to cluster...")
	if err := client.ApplyAll(manifests); err != nil {
		return fmt.Errorf("applying manifests: %w", err)
	}

	printSuccess("Resources created")

	// Wait for deployment
	printInfo("Waiting for server to be ready...")
	namespace := game.GetNamespace(serverCfg.Name)
	deployment := game.GetDeploymentName(serverCfg.Name)

	if err := client.WaitForReady(namespace, deployment, 180); err != nil {
		printWarning("Server is taking longer than expected to start")
		printInfo("Check logs with: zplay → View logs")
	} else {
		printSuccess("Server is ready!")
	}

	// Save state
	state.Add(config.ServerInfo{
		Name:       serverCfg.Name,
		Game:       game.Name(),
		Port:       serverCfg.Port,
		Memory:     serverCfg.Memory,
		MaxPlayers: serverCfg.MaxPlayers,
		CreatedAt:  time.Now().Format(time.RFC3339),
	})

	if err := config.SaveServerState(cfg, state); err != nil {
		printWarning("Could not save server state: " + err.Error())
	}

	fmt.Println()
	fmt.Println(successStyle.Render("═══════════════════════════════════════"))
	fmt.Printf("Connect to: %s:%d\n", cfg.Domain, serverCfg.Port)
	fmt.Println(successStyle.Render("═══════════════════════════════════════"))

	return nil
}

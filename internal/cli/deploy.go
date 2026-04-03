package cli

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/games/minecraft"
	"github.com/Zyrakk/zplay/internal/k8s"
	"github.com/Zyrakk/zplay/internal/util"
)

var (
	deployNameRegex   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,18}[a-z0-9]$`)
	deployMemoryRegex = regexp.MustCompile(`^\d+[GM]i$`)
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

	// Game-specific options
	switch game.Name() {
	case "terraria":
		fmt.Println("\nServer type:")
		fmt.Println("  1) Vanilla")
		fmt.Println("  2) tModLoader (mods)")
		fmt.Print("Choice [1]: ")
		variantChoice, _ := reader.ReadString('\n')
		variantChoice = strings.TrimSpace(variantChoice)

		switch variantChoice {
		case "", "1":
			serverCfg.Variant = "vanilla"
		case "2":
			serverCfg.Variant = "tmodloader"
			serverCfg.Memory = "4Gi"
			serverCfg.MemoryLimit = "8Gi"
		default:
			return fmt.Errorf("invalid server type selection")
		}
	case "minecraft":
		fmt.Println("\nServer type:")
		fmt.Println("  1) Vanilla")
		fmt.Println("  2) Paper (optimized)")
		fmt.Println("  3) Forge (mods)")
		fmt.Print("Choice [1]: ")
		variantChoice, _ := reader.ReadString('\n')
		variantChoice = strings.TrimSpace(variantChoice)

		switch variantChoice {
		case "", "1":
			serverCfg.Variant = "vanilla"
		case "2":
			serverCfg.Variant = "paper"
		case "3":
			serverCfg.Variant = "forge"
		default:
			return fmt.Errorf("invalid server type selection")
		}

		fmt.Print("\nMinecraft version (latest): ")
		mcVersion, _ := reader.ReadString('\n')
		mcVersion = strings.TrimSpace(mcVersion)
		if mcVersion != "" {
			serverCfg.Version = mcVersion
		}

		fmt.Print("MOTD (optional): ")
		motd, _ := reader.ReadString('\n')
		serverCfg.MOTD = strings.TrimSpace(motd)

		fmt.Print("Operators (comma-separated usernames, optional): ")
		ops, _ := reader.ReadString('\n')
		serverCfg.Ops = strings.TrimSpace(ops)

		// Server properties (optional)
		fmt.Println("\n" + dimStyle.Render("Server properties (all optional, Enter to skip):"))

		fmt.Print("  Difficulty [peaceful/easy/normal/hard]: ")
		mcDifficulty, _ := reader.ReadString('\n')
		mcDifficulty = strings.TrimSpace(mcDifficulty)
		if mcDifficulty != "" {
			serverCfg.Difficulty = mcDifficulty
		} else {
			serverCfg.Difficulty = ""
		}

		fmt.Print("  Gamemode [survival/creative/adventure/spectator]: ")
		gamemode, _ := reader.ReadString('\n')
		serverCfg.Gamemode = strings.TrimSpace(gamemode)

		fmt.Print("  World seed: ")
		seed, _ := reader.ReadString('\n')
		serverCfg.Seed = strings.TrimSpace(seed)

		fmt.Print("  PvP [true/false]: ")
		pvp, _ := reader.ReadString('\n')
		serverCfg.PvP = strings.TrimSpace(pvp)

		fmt.Print("  View distance [2-32]: ")
		viewDist, _ := reader.ReadString('\n')
		serverCfg.ViewDistance = strings.TrimSpace(viewDist)

		fmt.Print("  Level name: ")
		levelName, _ := reader.ReadString('\n')
		serverCfg.LevelName = strings.TrimSpace(levelName)

		fmt.Print("\n  Custom world (path, URL, or Enter to skip): ")
		worldSource, _ := reader.ReadString('\n')
		worldSource = strings.TrimSpace(worldSource)
		if worldSource != "" {
			worldDir, cleanup, err := resolveWorldSource(worldSource)
			if err != nil {
				return fmt.Errorf("invalid world source: %w", err)
			}
			if cleanup != nil {
				defer cleanup()
			}
			serverCfg.WorldSource = worldDir
			printSuccess(fmt.Sprintf("World validated: %s", worldDir))
		}
	}

	// Node selection
	if game.Name() == "terraria" && serverCfg.Variant == "tmodloader" {
		fmt.Println("\nSelect node:")
		fmt.Println("  1) lake (16GB RAM, x86) - required for tModLoader")
		fmt.Print("Choice [1]: ")
		nodeChoice, _ := reader.ReadString('\n')
		nodeChoice = strings.TrimSpace(nodeChoice)
		switch nodeChoice {
		case "", "1":
			serverCfg.NodeSelector = "lake"
		default:
			return fmt.Errorf("invalid node selection")
		}
	} else {
		fmt.Println("\nSelect node:")
		fmt.Println("  1) oracle1 (24GB RAM) - recommended")
		fmt.Println("  2) oracle2 (24GB RAM)")
		fmt.Println("  3) raspberry (8GB RAM) - light servers only")
		fmt.Println("  4) Auto (scheduler decides)")
		fmt.Print("Choice [1]: ")
		nodeChoice, _ := reader.ReadString('\n')
		nodeChoice = strings.TrimSpace(nodeChoice)

		switch nodeChoice {
		case "1":
			serverCfg.NodeSelector = "oracle1"
		case "2":
			serverCfg.NodeSelector = "oracle2"
		case "3":
			serverCfg.NodeSelector = "raspberry"
		case "", "4":
			serverCfg.NodeSelector = ""
		default:
			return fmt.Errorf("invalid node selection")
		}
	}

	// Memory
	fmt.Printf("Memory [%s]: ", serverCfg.Memory)
	memory, _ := reader.ReadString('\n')
	memory = strings.TrimSpace(memory)
	memoryChanged := false
	if memory != "" {
		serverCfg.Memory = memory
		memoryChanged = true
	}

	// Memory limit (double the request by default when memory is changed)
	if memoryChanged {
		memVal := strings.TrimSuffix(serverCfg.Memory, "Gi")
		if memInt, err := strconv.Atoi(memVal); err == nil {
			serverCfg.MemoryLimit = fmt.Sprintf("%dGi", memInt*2)
		}
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

		fmt.Println("\nDifficulty:")
		fmt.Println("  1) Classic")
		fmt.Println("  2) Expert")
		fmt.Println("  3) Master")
		fmt.Println("  4) Journey")
		fmt.Print("Choice [1]: ")
		difficultyChoice, _ := reader.ReadString('\n')
		difficultyChoice = strings.TrimSpace(difficultyChoice)

		switch difficultyChoice {
		case "", "1":
			serverCfg.Difficulty = "0"
		case "2":
			serverCfg.Difficulty = "1"
		case "3":
			serverCfg.Difficulty = "2"
		case "4":
			serverCfg.Difficulty = "3"
		default:
			return fmt.Errorf("invalid difficulty selection")
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

	if err := validateDeployConfig(serverCfg, state, game); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate
	if err := game.Validate(serverCfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if gameSupportsAutoBackup(game) {
		fmt.Print("Enable daily auto-backup? [Y/n]: ")
		autoBackupChoice, _ := reader.ReadString('\n')
		autoBackupChoice = strings.TrimSpace(strings.ToLower(autoBackupChoice))
		serverCfg.AutoBackup = autoBackupChoice != "n" && autoBackupChoice != "no"
	} else {
		serverCfg.AutoBackup = false
	}

	// Confirm
	fmt.Println()
	fmt.Println(dimStyle.Render("─────────────────────────────"))
	fmt.Printf("Game:        %s\n", game.DisplayName())
	if serverCfg.Variant != "" {
		fmt.Printf("Type:        %s\n", serverCfg.Variant)
	}
	fmt.Printf("Server:      %s\n", serverCfg.Name)
	if serverCfg.NodeSelector != "" {
		fmt.Printf("Node:        %s\n", serverCfg.NodeSelector)
	} else {
		fmt.Printf("Node:        auto\n")
	}
	fmt.Printf("Memory:      %s (limit: %s)\n", serverCfg.Memory, serverCfg.MemoryLimit)
	fmt.Printf("Max Players: %d\n", serverCfg.MaxPlayers)
	fmt.Printf("Port:        %d\n", serverCfg.Port)
	if serverCfg.WorldSize != "" {
		fmt.Printf("World Size:  %s\n", serverCfg.WorldSize)
	}
	if serverCfg.Difficulty != "" {
		if serverCfg.Game == "terraria" {
			diffNames := map[string]string{"0": "Classic", "1": "Expert", "2": "Master", "3": "Journey"}
			fmt.Printf("Difficulty:  %s\n", diffNames[serverCfg.Difficulty])
		} else {
			fmt.Printf("Difficulty:  %s\n", serverCfg.Difficulty)
		}
	}
	if serverCfg.Gamemode != "" {
		fmt.Printf("Gamemode:    %s\n", serverCfg.Gamemode)
	}
	if serverCfg.Seed != "" {
		fmt.Printf("Seed:        %s\n", serverCfg.Seed)
	}
	if serverCfg.PvP != "" {
		fmt.Printf("PvP:         %s\n", serverCfg.PvP)
	}
	if serverCfg.ViewDistance != "" {
		fmt.Printf("View Dist:   %s\n", serverCfg.ViewDistance)
	}
	if serverCfg.LevelName != "" {
		fmt.Printf("Level Name:  %s\n", serverCfg.LevelName)
	}
	if serverCfg.WorldSource != "" {
		fmt.Printf("World:       %s\n", serverCfg.WorldSource)
	}
	if gameSupportsAutoBackup(game) {
		if serverCfg.AutoBackup {
			fmt.Printf("Auto Backup: enabled (daily 4:00 AM)\n")
		} else {
			fmt.Printf("Auto Backup: disabled\n")
		}
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

	client := k8s.NewClient(cfg.Kubeconfig)

	if err := validateNodeExists(client, serverCfg.NodeSelector); err != nil {
		return err
	}

	if serverCfg.WorldSource != "" {
		infra, workload := splitManifests(manifests)

		printInfo("Creating namespace and storage...")
		if err := client.ApplyAll(infra); err != nil {
			return fmt.Errorf("applying infrastructure: %w", err)
		}

		printInfo("Uploading world to server...")
		if err := deployWorldUpload(client, game, serverCfg); err != nil {
			namespace := game.GetNamespace(serverCfg.Name)
			client.DeleteNamespace(namespace)
			return fmt.Errorf("uploading world: %w", err)
		}

		printInfo("Applying server configuration...")
		if err := client.ApplyAll(workload); err != nil {
			return fmt.Errorf("applying workload: %w", err)
		}
	} else {
		printInfo("Applying to cluster...")
		if err := client.ApplyAll(manifests); err != nil {
			return fmt.Errorf("applying manifests: %w", err)
		}
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

	if serverCfg.Variant == "tmodloader" {
		podName := ""
		for i := 0; i < 6; i++ {
			resolvedPodName, _ := client.GetPodName(namespace, "app=zplay,server="+serverCfg.Name)
			if resolvedPodName != "" {
				podName = resolvedPodName
				break
			}
			time.Sleep(5 * time.Second)
		}
		if podName == "" {
			podName = "<pod-name>"
		}

		fmt.Println()
		fmt.Println(dimStyle.Render("─────────────────────────────────────"))
		fmt.Println(dimStyle.Render("To install mods:"))
		fmt.Println()
		fmt.Println(dimStyle.Render("  Method 1 — Workshop IDs (recommended):"))
		fmt.Println(dimStyle.Render("    Edit the deployment env vars to add Workshop mod IDs:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    kubectl set env deployment/%s-terraria -n zplay-%s \\", serverCfg.Name, serverCfg.Name)))
		fmt.Println(dimStyle.Render("      TMOD_AUTODOWNLOAD=\"2824688072,2824688804\" \\"))
		fmt.Println(dimStyle.Render("      TMOD_ENABLEDMODS=\"2824688072,2824688804\""))
		fmt.Println(dimStyle.Render("    The pod will restart and download mods automatically."))
		fmt.Println()
		fmt.Println(dimStyle.Render("  Method 2 — Manual copy:"))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    kubectl cp ./ModName.tmod zplay-%s/%s:/data/tModLoader/Mods/", serverCfg.Name, podName)))
		fmt.Println(dimStyle.Render(fmt.Sprintf("    Then restart: zplay stop %s && zplay start %s", serverCfg.Name, serverCfg.Name)))
		fmt.Println()
		fmt.Println(dimStyle.Render("  Mod data is stored in /data/ (persisted via PVC)."))
		// TODO: Add a dedicated `zplay mods` command to automate tModLoader mod management.
		fmt.Println(dimStyle.Render("─────────────────────────────────────"))
	}

	// Save state
	state.Add(config.ServerInfo{
		Name:       serverCfg.Name,
		Game:       game.Name(),
		Namespace:  namespace,
		Variant:    serverCfg.Variant,
		AutoBackup: serverCfg.AutoBackup,
		Node:       serverCfg.NodeSelector,
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

func validateDeployConfig(serverCfg *games.ServerConfig, state *config.ServerState, game games.Game) error {
	if !deployNameRegex.MatchString(serverCfg.Name) {
		return fmt.Errorf("server name must match RFC 1123: lowercase letters, numbers, hyphens, 2-20 chars, no leading/trailing hyphen")
	}

	allowedPorts := allowedEntrypointPorts(game.Name())
	if len(allowedPorts) > 0 && !containsInt(allowedPorts, serverCfg.Port) {
		return fmt.Errorf("port %d has no configured entrypoint for %s (available ports: %s)", serverCfg.Port, game.Name(), formatPorts(allowedPorts))
	}

	for _, srv := range state.Servers {
		if srv.Name != serverCfg.Name && srv.Port == serverCfg.Port {
			return fmt.Errorf("port %d is already in use by server '%s'", serverCfg.Port, srv.Name)
		}
	}

	if !deployMemoryRegex.MatchString(serverCfg.Memory) {
		return fmt.Errorf("memory must match ^\\d+[GM]i$ (example: 4Gi, 512Mi)")
	}

	if serverCfg.NodeSelector == "raspberry" {
		memoryMi, err := util.MemoryToMi(serverCfg.Memory)
		if err != nil {
			return err
		}
		if memoryMi > 4*1024 {
			return fmt.Errorf("raspberry node is limited to 4Gi maximum")
		}
	}

	return nil
}

func validateNodeExists(client *k8s.Client, nodeSelector string) error {
	if nodeSelector == "" {
		return nil
	}

	nodes, err := client.GetNodes()
	if err != nil {
		return fmt.Errorf("checking nodes: %w", err)
	}

	for _, node := range nodes {
		if node == nodeSelector {
			return nil
		}
	}

	return fmt.Errorf("node '%s' not found in cluster (available: %s)", nodeSelector, strings.Join(nodes, ", "))
}

func allowedEntrypointPorts(gameName string) []int {
	switch gameName {
	case "terraria":
		return []int{7777, 7778}
	case "minecraft":
		return []int{25565, 25566}
	default:
		return nil
	}
}

func gameSupportsAutoBackup(game games.Game) bool {
	switch game.Name() {
	case "terraria", "minecraft":
		return true
	default:
		return false
	}
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func formatPorts(ports []int) string {
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		values = append(values, strconv.Itoa(port))
	}
	return strings.Join(values, ", ")
}

func splitManifests(manifests []string) (infra []string, workload []string) {
	for _, m := range manifests {
		if strings.Contains(m, "kind: Namespace") || strings.Contains(m, "kind: PersistentVolumeClaim") {
			infra = append(infra, m)
		} else {
			workload = append(workload, m)
		}
	}
	return
}

func deployWorldUpload(client *k8s.Client, game games.Game, serverCfg *games.ServerConfig) error {
	mc, ok := game.(*minecraft.Minecraft)
	if !ok {
		return fmt.Errorf("world upload requires Minecraft game type")
	}

	jobCfg := &games.ServerConfig{
		Name: serverCfg.Name,
		Game: serverCfg.Game,
	}
	jobManifest, err := mc.RenderUploadJob(jobCfg)
	if err != nil {
		return fmt.Errorf("rendering upload job: %w", err)
	}

	namespace := game.GetNamespace(serverCfg.Name)
	jobName := serverCfg.Name + "-upload"

	if err := client.Apply(jobManifest); err != nil {
		return fmt.Errorf("creating upload job: %w", err)
	}

	jobSelector := fmt.Sprintf("job-name=%s", jobName)
	podName, err := client.WaitForPodRunning(namespace, jobSelector, 120)
	if err != nil {
		client.DeleteJob(namespace, jobName)
		return fmt.Errorf("upload pod not ready: %w", err)
	}

	targetName := "world"
	if serverCfg.LevelName != "" {
		targetName = serverCfg.LevelName
	}
	targetPath := fmt.Sprintf("/data/%s", targetName)

	if err := client.CopyToPod(namespace, podName, serverCfg.WorldSource, targetPath); err != nil {
		client.DeleteJob(namespace, jobName)
		return fmt.Errorf("copying world: %w", err)
	}

	if err := client.DeleteJob(namespace, jobName); err != nil {
		printWarning("Could not delete upload job: " + err.Error())
	}

	return nil
}


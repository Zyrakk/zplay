package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/games/minecraft"
	"github.com/Zyrakk/zplay/internal/games/world"
	"github.com/Zyrakk/zplay/internal/k8s"
)

type UploadWorldOptions struct {
	Name string
	Path string
	URL  string
	Yes  bool
}

func RunUploadWorld(cfg *config.Config, opts UploadWorldOptions) error {
	// Look up server
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	srv, err := findServerByName(state, opts.Name)
	if err != nil {
		return err
	}

	if srv.Game != "minecraft" {
		return fmt.Errorf("upload-world is only supported for Minecraft servers")
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	mc, ok := game.(*minecraft.Minecraft)
	if !ok {
		return fmt.Errorf("unexpected game type")
	}

	// Resolve source BEFORE touching the server
	source := opts.Path
	if opts.URL != "" {
		source = opts.URL
	}
	if source == "" {
		return fmt.Errorf("either --path or --url is required")
	}

	worldDir, cleanup, err := resolveWorldSource(source)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	fmt.Printf("World validated: %s\n", worldDir)

	// Determine target path
	targetName := "world"

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	client := k8s.NewClient(cfg.Kubeconfig)

	// Check server status
	replicas, _ := client.GetReplicas(namespace, deployment)
	if replicas > 0 && !opts.Yes {
		fmt.Printf("\nServer %q is currently running.\n", srv.Name)
		fmt.Println("This will stop the server, replace the world, and restart it.")
		fmt.Print("Continue? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Scale down if running
	wasRunning := replicas > 0
	if wasRunning {
		printInfo("Stopping server...")
		if err := client.ScaleDeployment(namespace, deployment, 0); err != nil {
			return fmt.Errorf("stopping server: %w", err)
		}
		printInfo("Waiting for server to stop...")
		// WaitForReady will error when there are 0 replicas - that's expected
		_ = client.WaitForReady(namespace, deployment, 60)
	}

	// Create upload job
	printInfo("Creating upload job...")
	serverCfg := &games.ServerConfig{
		Name: srv.Name,
		Game: srv.Game,
	}
	jobManifest, err := mc.RenderUploadJob(serverCfg)
	if err != nil {
		return fmt.Errorf("rendering upload job: %w", err)
	}

	if err := client.Apply(jobManifest); err != nil {
		return fmt.Errorf("applying upload job: %w", err)
	}

	// Wait for upload pod to be running
	printInfo("Waiting for upload pod...")
	jobSelector := fmt.Sprintf("job-name=%s-upload", srv.Name)
	podName, err := client.WaitForPodRunning(namespace, jobSelector, 120)
	if err != nil {
		client.DeleteJob(namespace, srv.Name+"-upload")
		return fmt.Errorf("upload pod not ready: %w", err)
	}

	// Clear existing world and copy new one
	targetPath := fmt.Sprintf("/data/%s", targetName)
	printInfo(fmt.Sprintf("Clearing existing world at %s...", targetPath))
	if err := client.ExecInPod(namespace, podName, []string{"rm", "-rf", targetPath}); err != nil {
		client.DeleteJob(namespace, srv.Name+"-upload")
		return fmt.Errorf("clearing world: %w", err)
	}

	printInfo("Uploading world...")
	if err := client.CopyToPod(namespace, podName, worldDir, targetPath); err != nil {
		client.DeleteJob(namespace, srv.Name+"-upload")
		return fmt.Errorf("copying world: %w", err)
	}

	// Cleanup upload job
	printInfo("Cleaning up upload job...")
	if err := client.DeleteJob(namespace, srv.Name+"-upload"); err != nil {
		printWarning("Could not delete upload job: " + err.Error())
	}

	// Restart if it was running
	if wasRunning {
		printInfo("Starting server...")
		if err := client.ScaleDeployment(namespace, deployment, 1); err != nil {
			return fmt.Errorf("starting server: %w", err)
		}

		printInfo("Waiting for server to be ready...")
		if err := client.WaitForReady(namespace, deployment, 300); err != nil {
			printWarning("Server is taking longer than expected to start")
		} else {
			printSuccess("Server is ready!")
		}
	} else {
		printSuccess("World uploaded. Start the server with: zplay start " + srv.Name)
	}

	fmt.Printf("\nConnect to: %s:%d\n", cfg.Domain, srv.Port)
	return nil
}

func RunUploadWorldInteractive(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Upload World"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load server state
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	// Filter to Minecraft servers only
	var mcServers []config.ServerInfo
	for _, srv := range state.Servers {
		if srv.Game == "minecraft" {
			mcServers = append(mcServers, srv)
		}
	}

	if len(mcServers) == 0 {
		fmt.Println("No Minecraft servers deployed.")
		return nil
	}

	// Select server
	fmt.Println("Select server:")
	for i, srv := range mcServers {
		fmt.Printf("  %d) %s (%s)\n", i+1, srv.Name, srv.Variant)
	}
	fmt.Print("\nChoice: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	idx := 0
	if choice != "" {
		parsed, err := fmt.Sscanf(choice, "%d", &idx)
		if parsed != 1 || err != nil || idx < 1 || idx > len(mcServers) {
			return fmt.Errorf("invalid selection")
		}
		idx--
	} else {
		return fmt.Errorf("server selection is required")
	}

	srv := mcServers[idx]

	// Get source
	fmt.Print("\nWorld source (local path or URL): ")
	source, _ := reader.ReadString('\n')
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("world source is required")
	}

	opts := UploadWorldOptions{
		Name: srv.Name,
		Yes:  false,
	}
	if world.IsURL(source) {
		opts.URL = source
	} else {
		opts.Path = source
	}

	return RunUploadWorld(cfg, opts)
}

// resolveWorldSource resolves a local path or URL to a validated world directory.
// Returns the world directory path and an optional cleanup function.
func resolveWorldSource(source string) (string, func(), error) {
	if world.IsURL(source) {
		printInfo("Downloading world...")
		downloaded, err := world.Download(source)
		if err != nil {
			return "", nil, fmt.Errorf("downloading world: %w", err)
		}

		worldDir, err := world.ResolveLocal(downloaded)
		if err != nil {
			os.Remove(downloaded)
			return "", nil, err
		}

		cleanup := func() {
			os.Remove(downloaded)
		}
		return worldDir, cleanup, nil
	}

	worldDir, err := world.ResolveLocal(source)
	if err != nil {
		return "", nil, err
	}
	return worldDir, nil, nil
}

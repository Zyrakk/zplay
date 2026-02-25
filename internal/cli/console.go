package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunConsole(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Server Console"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers available."))
		return nil
	}

	// List servers
	fmt.Println("Select server:")
	for i, srv := range state.Servers {
		fmt.Printf("  %d) %s (%s)\n", i+1, srv.Name, srv.Game)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choice: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "" {
		return nil
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(state.Servers) {
		return fmt.Errorf("invalid selection")
	}

	srv := state.Servers[idx-1]

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	client := k8s.NewClient(cfg.Kubeconfig)

	fmt.Println()

	hasTmux := tmuxAvailable()
	if hasTmux {
		printInfo(fmt.Sprintf("Attaching to %s console via tmux...", srv.Name))
		printInfo("Press Ctrl+B then D to detach (server keeps running)")
		fmt.Println()

		if err := client.AttachConsoleViaTmux(namespace, deployment); err != nil {
			printWarning("tmux attach failed: " + err.Error())
			printInfo("Falling back to direct attach...")
			fmt.Println()
		} else {
			return nil
		}
	}

	printInfo(fmt.Sprintf("Attaching to %s console...", srv.Name))
	if !hasTmux {
		printWarning("tmux not found - using direct attach. Ctrl+C will stop the server safely.")
		printInfo("Install tmux for clean detach support.")
	} else {
		printWarning("Using direct attach fallback. Ctrl+C will stop the server safely.")
	}
	fmt.Println()

	if err := client.AttachConsole(namespace, deployment); err != nil {
		printWarning("console attach failed: " + err.Error())
		printInfo("Falling back to live deployment logs...")
		return client.Logs(namespace, deployment, true)
	}
	return nil
}

func tmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

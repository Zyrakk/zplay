package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunLogs(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Server Logs"))
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

	// Follow option
	fmt.Print("Follow logs? [Y/n]: ")
	followChoice, _ := reader.ReadString('\n')
	followChoice = strings.TrimSpace(strings.ToLower(followChoice))
	follow := followChoice != "n" && followChoice != "no"

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)

	fmt.Println()
	if follow {
		printInfo("Streaming logs (Ctrl+C to stop)...")
	} else {
		printInfo("Fetching logs...")
	}
	fmt.Println()

	client := k8s.NewClient(cfg.Kubeconfig)
	return client.Logs(namespace, deployment, follow)
}

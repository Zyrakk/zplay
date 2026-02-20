package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunList(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Server List"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	client := k8s.NewClient(cfg.Kubeconfig)
	reader := bufio.NewReader(os.Stdin)

	discovered, err := client.DiscoverServers()
	if err != nil {
		printWarning("Could not reconcile local state with cluster: " + err.Error())
	} else {
		discoveredByName := make(map[string]k8s.DiscoveredServer, len(discovered))
		for _, srv := range discovered {
			discoveredByName[srv.Name] = srv
		}

		// TODO: Support non-interactive reconciliation mode via `zplay list --sync`.
		added, orphaned := config.Reconcile(state, discovered)
		stateChanged := false

		if len(added) > 0 {
			fmt.Println(warningStyle.Render(fmt.Sprintf("⚠ Found %d server(s) in cluster not tracked locally:", len(added))))
			for _, name := range added {
				if srv, ok := discoveredByName[name]; ok {
					fmt.Printf("  - %s (%s) in namespace %s\n", srv.Name, srv.Game, srv.Namespace)
				}
			}

			if promptYesNo(reader, "Adopt into local state? [Y/n]: ") {
				adopted := 0
				for _, name := range added {
					if state.Get(name) != nil {
						continue
					}
					srv, ok := discoveredByName[name]
					if !ok {
						continue
					}

					state.Add(serverInfoFromDiscovered(state, srv))
					adopted++
				}

				if adopted > 0 {
					stateChanged = true
					printSuccess(fmt.Sprintf("Adopted %d server(s) into local state.", adopted))
				}
			}

			fmt.Println()
		}

		if len(orphaned) > 0 {
			fmt.Println(dimStyle.Render(fmt.Sprintf("ℹ %d server(s) in local state not found in cluster (removed):", len(orphaned))))
			for _, name := range orphaned {
				fmt.Printf("  - %s\n", name)
			}

			if promptYesNo(reader, "Clean from local state? [Y/n]: ") {
				removed := 0
				for _, name := range orphaned {
					if state.Remove(name) {
						removed++
					}
				}
				if removed > 0 {
					stateChanged = true
					printSuccess(fmt.Sprintf("Removed %d server(s) from local state.", removed))
				}
			}

			fmt.Println()
		}

		if stateChanged {
			if err := config.SaveServerState(cfg, state); err != nil {
				printWarning("Could not save reconciled server state: " + err.Error())
			}
		}
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers deployed yet."))
		fmt.Println(dimStyle.Render("Use 'Deploy server' to create one."))
		return nil
	}

	// Header
	fmt.Printf("%-15s %-14s %-12s %-8s %-10s %-10s %s\n",
		"NAME", "GAME", "NODE", "PORT", "MEMORY", "STATUS", "ADDRESS")
	fmt.Println(dimStyle.Render("──────────────────────────────────────────────────────────────────────────────────────"))

	for _, srv := range state.Servers {
		game := games.Get(srv.Game)
		if game == nil {
			continue
		}

		namespace := srv.Namespace
		if namespace == "" {
			namespace = game.GetNamespace(srv.Name)
		}
		deployment := game.GetDeploymentName(srv.Name)
		status, _ := client.GetPodStatus(namespace, fmt.Sprintf("app=zplay,server=%s,!job-name", srv.Name))
		if status == "" {
			status = "Unknown"
		}
		replicas, err := client.GetReplicas(namespace, deployment)
		if err == nil && replicas == 0 {
			status = "Stopped"
		}

		statusStyle := dimStyle
		switch status {
		case "Running":
			statusStyle = successStyle
		case "Pending":
			statusStyle = warningStyle
		case "Stopped":
			statusStyle = warningStyle
		case "Failed", "Unknown":
			statusStyle = errorStyle
		}

		address := "n/a"
		if srv.Port > 0 {
			address = fmt.Sprintf("%s:%d", cfg.Domain, srv.Port)
		}
		node := srv.Node
		if node == "" {
			node = "auto"
		}
		memory := srv.Memory
		if memory == "" {
			memory = "-"
		}
		gameLabel := srv.Game
		if srv.Game == "terraria" && srv.Variant == "tmodloader" {
			gameLabel = "terraria-tmod"
		}

		fmt.Printf("%-15s %-14s %-12s %-8d %-10s %-10s %s\n",
			srv.Name,
			gameLabel,
			node,
			srv.Port,
			memory,
			statusStyle.Render(status),
			address,
		)
	}

	fmt.Println()
	return nil
}

func promptYesNo(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt)
	choice, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(choice) == "" {
		return false
	}
	choice = strings.TrimSpace(strings.ToLower(choice))
	return choice != "n" && choice != "no"
}

func serverInfoFromDiscovered(state *config.ServerState, discovered k8s.DiscoveredServer) config.ServerInfo {
	port := 0
	if game := games.Get(discovered.Game); game != nil {
		port = state.NextPort(discovered.Game, game.DefaultPort())
	}

	variant := ""
	if discovered.Game == "terraria" {
		variant = "vanilla"
	}

	return config.ServerInfo{
		Name:       discovered.Name,
		Game:       discovered.Game,
		Namespace:  discovered.Namespace,
		Variant:    variant,
		AutoBackup: false,
		Node:       "",
		Port:       port,
		Memory:     "4Gi",
		MaxPlayers: 8,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
}

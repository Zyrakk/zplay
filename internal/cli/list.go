package cli

import (
	"fmt"

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

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers deployed yet."))
		fmt.Println(dimStyle.Render("Use 'Deploy server' to create one."))
		return nil
	}

	client := k8s.NewClient(cfg)

	// Header
	fmt.Printf("%-15s %-14s %-12s %-8s %-10s %-10s %s\n",
		"NAME", "GAME", "NODE", "PORT", "MEMORY", "STATUS", "ADDRESS")
	fmt.Println(dimStyle.Render("──────────────────────────────────────────────────────────────────────────────────────"))

	for _, srv := range state.Servers {
		game := games.Get(srv.Game)
		if game == nil {
			continue
		}

		namespace := game.GetNamespace(srv.Name)
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

		address := fmt.Sprintf("%s:%d", cfg.Domain, srv.Port)
		node := srv.Node
		if node == "" {
			node = "auto"
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
			srv.Memory,
			statusStyle.Render(status),
			address,
		)
	}

	fmt.Println()
	return nil
}

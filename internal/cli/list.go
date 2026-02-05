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
	fmt.Printf("%-15s %-12s %-8s %-10s %-10s %s\n",
		"NAME", "GAME", "PORT", "MEMORY", "STATUS", "ADDRESS")
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────────────────"))

	for _, srv := range state.Servers {
		game := games.Get(srv.Game)
		if game == nil {
			continue
		}

		namespace := game.GetNamespace(srv.Name)
		status, _ := client.GetPodStatus(namespace)
		if status == "" {
			status = "Unknown"
		}

		statusStyle := dimStyle
		switch status {
		case "Running":
			statusStyle = successStyle
		case "Pending":
			statusStyle = warningStyle
		case "Failed", "Unknown":
			statusStyle = errorStyle
		}

		address := fmt.Sprintf("%s:%d", cfg.Domain, srv.Port)

		fmt.Printf("%-15s %-12s %-8d %-10s %-10s %s\n",
			srv.Name,
			srv.Game,
			srv.Port,
			srv.Memory,
			statusStyle.Render(status),
			address,
		)
	}

	fmt.Println()
	return nil
}

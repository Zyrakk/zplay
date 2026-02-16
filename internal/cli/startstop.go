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

func RunStartStop(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Start/Stop Server"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers available."))
		return nil
	}

	client := k8s.NewClient(cfg)

	fmt.Println("Select server:")
	for i, srv := range state.Servers {
		status := "Unknown"
		game := games.Get(srv.Game)
		if game != nil {
			namespace := game.GetNamespace(srv.Name)
			deployment := game.GetDeploymentName(srv.Name)
			replicas, err := client.GetReplicas(namespace, deployment)
			if err == nil {
				if replicas > 0 {
					status = "Running"
				} else {
					status = "Stopped"
				}
			}
		}
		fmt.Printf("  %d) %s (%s) - %s\n", i+1, srv.Name, srv.Game, status)
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

	namespace := game.GetNamespace(srv.Name)
	deployment := game.GetDeploymentName(srv.Name)

	replicas, err := client.GetReplicas(namespace, deployment)
	if err != nil {
		return fmt.Errorf("getting replicas: %w", err)
	}

	if replicas > 0 {
		fmt.Printf("Stop server '%s'? [Y/n]: ", srv.Name)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm == "n" || confirm == "no" {
			fmt.Println("Cancelled.")
			return nil
		}

		printInfo("Stopping server...")
		if err := client.ScaleDeployment(namespace, deployment, 0); err != nil {
			return fmt.Errorf("stopping server: %w", err)
		}

		printSuccess(fmt.Sprintf("Server '%s' stopped", srv.Name))
		return nil
	}

	fmt.Printf("Start server '%s'? [Y/n]: ", srv.Name)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelled.")
		return nil
	}

	printInfo("Starting server...")
	if err := client.ScaleDeployment(namespace, deployment, 1); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	printInfo("Waiting for server to be ready...")
	if err := client.WaitForReady(namespace, deployment, 180); err != nil {
		printWarning("Server is taking longer than expected to start")
	}

	printSuccess(fmt.Sprintf("Server '%s' started", srv.Name))
	return nil
}

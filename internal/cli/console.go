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

	namespace := game.GetNamespace(srv.Name)
	deployment := game.GetDeploymentName(srv.Name)

	fmt.Println()
	printInfo(fmt.Sprintf("Attaching to %s console...", srv.Name))
	printInfo("Press Ctrl+P, Ctrl+Q to detach without stopping the server")
	fmt.Println()

	client := k8s.NewClient(cfg)
	return client.AttachConsole(namespace, deployment)
}

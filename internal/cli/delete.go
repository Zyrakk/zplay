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

func RunDelete(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Delete Server"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers to delete."))
		return nil
	}

	// List servers
	fmt.Println("Select server to delete:")
	for i, srv := range state.Servers {
		fmt.Printf("  %d) %s (%s, port %d)\n", i+1, srv.Name, srv.Game, srv.Port)
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

	// Confirm
	fmt.Println()
	printWarning(fmt.Sprintf("This will permanently delete server '%s' and all its data!", srv.Name))
	fmt.Print("\nType the server name to confirm: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)

	if confirm != srv.Name {
		fmt.Println("Cancelled.")
		return nil
	}

	// Get game for namespace
	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := game.GetNamespace(srv.Name)

	// Delete
	fmt.Println()
	printInfo("Deleting namespace " + namespace + "...")

	client := k8s.NewClient(cfg)
	if err := client.DeleteNamespace(namespace); err != nil {
		return fmt.Errorf("deleting namespace: %w", err)
	}

	// Update state
	state.Remove(srv.Name)
	if err := config.SaveServerState(cfg, state); err != nil {
		printWarning("Could not update server state: " + err.Error())
	}

	printSuccess(fmt.Sprintf("Server '%s' deleted", srv.Name))

	return nil
}

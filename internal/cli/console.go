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
	client := k8s.NewClient(cfg)

	if srv.Variant == "tmodloader" {
		fmt.Println()
		fmt.Println("Console type:")
		fmt.Println("  1) Send commands (via inject)")
		fmt.Println("  2) View live output (tmux attach)")
		fmt.Print("Choice [1]: ")
		consoleChoice, _ := reader.ReadString('\n')
		consoleChoice = strings.TrimSpace(consoleChoice)

		if consoleChoice == "2" {
			fmt.Println()
			printInfo("Attaching to tmux session... Press Ctrl+B then D to detach.")
			fmt.Println()
			if err := client.Exec(namespace, deployment, []string{"tmux", "attach"}); err != nil {
				printWarning("tmux attach failed: " + err.Error())
				printInfo("Falling back to live deployment logs...")
				return client.Logs(namespace, deployment, true)
			}
			return nil
		}

		fmt.Println()
		printInfo(fmt.Sprintf("Connected to %s tModLoader console.", srv.Name))
		printInfo("Type commands and press Enter. Type 'exit' to quit.")
		fmt.Println()

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			command := strings.TrimSpace(scanner.Text())
			if command == "" {
				continue
			}
			if command == "exit" || command == "quit" {
				break
			}

			if err := client.ExecNoTTY(namespace, deployment, []string{"inject", command}); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
		return nil
	}

	fmt.Println()
	printInfo(fmt.Sprintf("Attaching to %s console...", srv.Name))
	printInfo("Press Ctrl+P, Ctrl+Q to detach without stopping the server")
	fmt.Println()

	if err := client.AttachConsole(namespace, deployment); err != nil {
		printWarning("console attach failed: " + err.Error())
		printInfo("Falling back to live deployment logs...")
		return client.Logs(namespace, deployment, true)
	}
	return nil
}

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunBackup(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Backup Server"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers available."))
		return nil
	}

	fmt.Println("Select server to backup:")
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

	fmt.Printf("Backup server '%s'? [Y/n]: ", srv.Name)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelled.")
		return nil
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	timestamp := time.Now().Format("20060102-150405")
	serverCfg := &games.ServerConfig{
		Name:         srv.Name,
		Timestamp:    timestamp,
		Game:         srv.Game,
		Variant:      srv.Variant,
		Memory:       srv.Memory,
		MaxPlayers:   srv.MaxPlayers,
		Port:         srv.Port,
		NodeSelector: srv.Node,
		Domain:       cfg.Domain,
	}
	serverCfg.BackupPath = cfg.Backup.Path
	serverCfg.BackupNode = cfg.Backup.Node

	jobManifest, err := game.RenderBackupJob(serverCfg)
	if err != nil {
		return fmt.Errorf("rendering backup job: %w", err)
	}

	namespace := resolveNamespace(srv, game)
	jobName := fmt.Sprintf("%s-backup-%s", srv.Name, timestamp)
	client := k8s.NewClient(cfg.Kubeconfig)

	// Save world before backup
	if podName, podErr := client.GetPodName(namespace, fmt.Sprintf("app=zplay,server=%s,!job-name", srv.Name)); podErr == nil && podName != "" {
		printInfo("Saving world before backup...")
		if saveErr := client.SaveWorld(namespace, podName, 30); saveErr != nil {
			printWarning("Could not save world before backup: " + saveErr.Error())
		}
	}

	printInfo("Creating backup job...")
	if err := client.RunBackupJob(namespace, jobName, jobManifest, 120); err != nil {
		printError("Backup failed: " + err.Error())
		return nil
	}

	printSuccess(fmt.Sprintf("Backup created: /mnt/das/zplay-backups/%s-%s.tar.gz", srv.Name, timestamp))
	return nil
}

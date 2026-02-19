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

func RunRestore(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Restore Backup"))
	fmt.Println()

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		fmt.Println(dimStyle.Render("No servers available."))
		return nil
	}

	fmt.Println("Select server to restore:")
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

	printInfo("Listing available backups...")
	listJobSuffix := time.Now().Format("20060102-150405")
	listJobName := fmt.Sprintf("%s-backup-list-%s", srv.Name, listJobSuffix)
	listManifest := renderBackupListJobManifest(srv.Name, srv.Game, listJobName)

	listOutput, err := client.RunJobAndGetLogs(namespace, listJobName, listManifest, 90)
	if err != nil {
		return fmt.Errorf("listing backups: %w", err)
	}

	backups := parseBackupFilenames(listOutput)
	if len(backups) == 0 {
		fmt.Println(dimStyle.Render("No backups found for this server in /mnt/das/zplay-backups/."))
		return nil
	}

	fmt.Println()
	fmt.Println("Select backup to restore:")
	for i, backupFile := range backups {
		fmt.Printf("  %d) %s\n", i+1, backupFile)
	}
	fmt.Println()

	fmt.Print("Choice: ")
	backupChoice, _ := reader.ReadString('\n')
	backupChoice = strings.TrimSpace(backupChoice)
	if backupChoice == "" {
		return nil
	}

	backupIdx, err := strconv.Atoi(backupChoice)
	if err != nil || backupIdx < 1 || backupIdx > len(backups) {
		return fmt.Errorf("invalid backup selection")
	}
	backupFile := backups[backupIdx-1]

	fmt.Println()
	printWarning("This will overwrite current server data!")
	fmt.Print("Type server name to confirm: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if confirm != srv.Name {
		fmt.Println("Cancelled.")
		return nil
	}

	restoreTimestamp := time.Now().Format("20060102-150405")
	serverCfg := &games.ServerConfig{
		Name:         srv.Name,
		Timestamp:    restoreTimestamp,
		BackupFile:   backupFile,
		Game:         srv.Game,
		Variant:      srv.Variant,
		AutoBackup:   srv.AutoBackup,
		Memory:       srv.Memory,
		MaxPlayers:   srv.MaxPlayers,
		Port:         srv.Port,
		NodeSelector: srv.Node,
		Domain:       cfg.Domain,
	}

	restoreManifest, err := game.RenderRestoreJob(serverCfg)
	if err != nil {
		return fmt.Errorf("rendering restore job: %w", err)
	}
	restoreJobName := fmt.Sprintf("%s-restore-%s", srv.Name, restoreTimestamp)

	printInfo("Stopping server...")
	if err := client.ScaleDeployment(namespace, deployment, 0); err != nil {
		return fmt.Errorf("stopping server: %w", err)
	}

	printInfo("Running restore job...")
	if err := client.RunJob(namespace, restoreJobName, restoreManifest, 180); err != nil {
		printError("Restore failed: " + err.Error())
		printWarning("Server remains stopped for safety.")
		return nil
	}

	printInfo("Starting server...")
	if err := client.ScaleDeployment(namespace, deployment, 1); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	printInfo("Waiting for server to be ready...")
	if err := client.WaitForReady(namespace, deployment, 180); err != nil {
		printWarning("Server is taking longer than expected to start")
		printInfo("Check logs with: zplay → View logs")
	} else {
		printSuccess("Server is ready!")
	}

	printSuccess(fmt.Sprintf("Restore completed from backup: %s", backupFile))
	return nil
}

func renderBackupListJobManifest(serverName, gameName, jobName string) string {
	return fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: zplay-%s
  labels:
    app: zplay
    game: %s
    server: %s
    type: backup-list
spec:
  ttlSecondsAfterFinished: 120
  backoffLimit: 1
  template:
    spec:
      restartPolicy: Never
      nodeSelector:
        kubernetes.io/hostname: oracle1
      containers:
        - name: backup-list
          image: alpine:3.19
          command: ["/bin/sh", "-c"]
          args:
            - |
              (ls -1t /backup/%s-*.tar.gz 2>/dev/null || true) | sed 's|.*/||'
          volumeMounts:
            - name: backup
              mountPath: /backup
      volumes:
        - name: backup
          hostPath:
            path: /mnt/das/zplay-backups
            type: DirectoryOrCreate
`, jobName, serverName, gameName, serverName, serverName)
}

func parseBackupFilenames(output string) []string {
	lines := strings.Split(output, "\n")
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		results = append(results, trimmed)
	}
	return results
}

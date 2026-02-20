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

func RunStatus(cfg *config.Config, serverName string) error {
	interactive := strings.TrimSpace(serverName) == ""
	if interactive {
		clearScreen()
		fmt.Println(titleStyle.Render("Server Status"))
		fmt.Println()
	}

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	if len(state.Servers) == 0 {
		if interactive {
			fmt.Println(dimStyle.Render("No servers available."))
			return nil
		}
		return fmt.Errorf("no servers available")
	}

	targetName := strings.TrimSpace(serverName)
	if targetName == "" {
		selected, err := selectServerForStatus(state)
		if err != nil {
			return err
		}
		if selected == "" {
			return nil
		}
		targetName = selected
	}

	srv, err := findServerByName(state, targetName)
	if err != nil {
		return err
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	labelSelector := fmt.Sprintf("app=zplay,server=%s,!job-name", srv.Name)
	client := k8s.NewClient(cfg.Kubeconfig)

	status := resolveServerStatus(srv, client)
	node := "N/A"
	if nodeName, err := client.GetPodNodeName(namespace, labelSelector); err == nil && nodeName != "" {
		node = nodeName
	}

	uptime := "N/A"
	if startTime, err := client.GetPodStartTime(namespace, labelSelector); err == nil && startTime != "" {
		uptime = formatUptime(startTime)
	}

	address := "N/A"
	if srv.Port > 0 {
		address = fmt.Sprintf("%s:%d", cfg.Domain, srv.Port)
	}

	created := formatTimestampDisplay(srv.CreatedAt)
	gameDisplay := formatGameWithVariant(game, srv)

	resources, _ := client.GetDeploymentResources(namespace, deployment)

	memoryRequest := firstNonEmpty(resources.MemoryRequest, srv.Memory, "N/A")
	memoryLimitFallback := ""
	if srv.Memory != "" {
		if inferred, err := inferMemoryLimit(srv.Memory); err == nil {
			memoryLimitFallback = inferred
		}
	}
	memoryLimit := firstNonEmpty(resources.MemoryLimit, memoryLimitFallback, "N/A")

	memoryUsage := "N/A"
	cpuUsage := "N/A"
	if cpuRaw, memoryRaw, err := client.GetPodTop(namespace, labelSelector); err == nil {
		if formatted := formatCPUCores(cpuRaw); formatted != "" {
			cpuUsage = formatted
		}
		if formatted := formatMemoryGi(memoryRaw); formatted != "" {
			memoryUsage = formatted
		}
	}

	cpuLimit := "N/A"
	if formatted := formatCPUCores(resources.CPULimit); formatted != "" {
		cpuLimit = formatted
	}

	pvcLine := "N/A"
	if pvcInfo, err := client.GetPVCInfo(namespace); err == nil {
		storage := firstNonEmpty(pvcInfo.StorageRequest, "N/A")
		if storage != "N/A" {
			if pvcInfo.StorageClass != "" {
				pvcLine = fmt.Sprintf("%s (%s)", storage, pvcInfo.StorageClass)
			} else {
				pvcLine = storage
			}
		}
	}

	autoBackup := "N/A"
	if enabled, err := client.HasBackupCronJob(namespace); err == nil {
		if enabled {
			autoBackup = "Enabled (daily 4:00 AM)"
		} else {
			autoBackup = "Disabled"
		}
	}

	lastBackup := "N/A"
	if ts, err := client.GetLastBackupTimestamp(namespace); err == nil && ts != "" {
		lastBackup = formatTimestampDisplay(ts)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Server:      %s\n", srv.Name)
	fmt.Printf("Game:        %s\n", gameDisplay)
	fmt.Printf("Status:      %s\n", status)
	fmt.Printf("Node:        %s\n", node)
	fmt.Printf("Uptime:      %s\n", uptime)
	fmt.Printf("Port:        %s\n", address)
	fmt.Printf("Created:     %s\n", created)
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("Resources:")
	fmt.Printf("  Memory:    %s / %s (request) / %s (limit)\n", memoryUsage, memoryRequest, memoryLimit)
	fmt.Printf("  CPU:       %s / %s cores\n", cpuUsage, cpuLimit)
	fmt.Println("Storage:")
	fmt.Printf("  PVC:       %s\n", pvcLine)
	fmt.Println("Backup:")
	fmt.Printf("  Auto:      %s\n", autoBackup)
	fmt.Printf("  Last:      %s\n", lastBackup)
	fmt.Println("═══════════════════════════════════════")

	return nil
}

func selectServerForStatus(state *config.ServerState) (string, error) {
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
		return "", nil
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(state.Servers) {
		return "", fmt.Errorf("invalid selection")
	}

	return state.Servers[idx-1].Name, nil
}

func formatGameWithVariant(game games.Game, srv config.ServerInfo) string {
	gameName := srv.Game
	if game != nil {
		gameName = game.DisplayName()
	}

	variant := strings.TrimSpace(srv.Variant)
	if variant == "" && srv.Game == "terraria" {
		variant = "vanilla"
	}
	if variant == "" {
		variant = "default"
	}

	return fmt.Sprintf("%s (%s)", gameName, variant)
}

func formatTimestampDisplay(timestamp string) string {
	value := strings.TrimSpace(timestamp)
	if value == "" {
		return "N/A"
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format("2006-01-02 15:04:05")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.Format("2006-01-02 15:04:05")
	}

	return value
}

func formatUptime(startTime string) string {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(startTime))
	if err != nil {
		return "N/A"
	}

	elapsed := time.Since(parsed)
	if elapsed < 0 {
		return "N/A"
	}

	days := int(elapsed.Hours()) / 24
	hours := int(elapsed.Hours()) % 24
	minutes := int(elapsed.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatCPUCores(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || value == "<no value>" {
		return ""
	}

	if strings.HasSuffix(value, "m") {
		milli, err := strconv.ParseFloat(strings.TrimSuffix(value, "m"), 64)
		if err != nil {
			return value
		}
		return trimFloat(milli / 1000)
	}

	cores, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return trimFloat(cores)
}

func formatMemoryGi(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || value == "<no value>" {
		return ""
	}

	type unitScale struct {
		suffix string
		scale  float64
	}

	scales := []unitScale{
		{suffix: "Ki", scale: 1.0 / (1024 * 1024)},
		{suffix: "Mi", scale: 1.0 / 1024},
		{suffix: "Gi", scale: 1},
		{suffix: "Ti", scale: 1024},
	}

	for _, entry := range scales {
		if strings.HasSuffix(value, entry.suffix) {
			parsed, err := strconv.ParseFloat(strings.TrimSuffix(value, entry.suffix), 64)
			if err != nil {
				return value
			}
			return fmt.Sprintf("%sGi", trimFloat(parsed*entry.scale))
		}
	}

	return value
}

func trimFloat(value float64) string {
	formatted := fmt.Sprintf("%.1f", value)
	return strings.TrimSuffix(formatted, ".0")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

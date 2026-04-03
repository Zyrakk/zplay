package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
	"github.com/Zyrakk/zplay/internal/util"
)

type DeployOptions struct {
	Game       string
	Variant    string
	Name       string
	Memory     string
	Node       string
	Port       int
	Password   string
	MaxPlayers int
	WorldSize  string
	Difficulty   string
	AutoBackup   bool
	Gamemode     string
	Seed         string
	PvP          string
	ViewDistance string
	LevelName    string
}

type listServerJSON struct {
	Name    string `json:"name"`
	Game    string `json:"game"`
	Variant string `json:"variant,omitempty"`
	Port    int    `json:"port"`
	Node    string `json:"node"`
	Status  string `json:"status"`
}

func RunDeployNonInteractive(cfg *config.Config, opts DeployOptions) error {
	gameName := strings.TrimSpace(opts.Game)
	game := games.Get(gameName)
	if game == nil {
		return fmt.Errorf("unknown game: %s", gameName)
	}

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return fmt.Errorf("server name is required")
	}
	if state.Get(name) != nil {
		return fmt.Errorf("server '%s' already exists", name)
	}
	if opts.Port <= 0 {
		return fmt.Errorf("port must be greater than 0")
	}

	serverCfg := games.NewServerConfig(cfg)
	serverCfg.Game = game.Name()
	serverCfg.Name = name
	serverCfg.Variant = strings.TrimSpace(opts.Variant)
	serverCfg.Memory = strings.TrimSpace(opts.Memory)
	serverCfg.Port = opts.Port
	serverCfg.Password = strings.TrimSpace(opts.Password)
	serverCfg.MaxPlayers = opts.MaxPlayers
	serverCfg.WorldSize = strings.TrimSpace(opts.WorldSize)
	serverCfg.Difficulty = strings.TrimSpace(opts.Difficulty)
	// Terraria defaults to Classic (0) if not specified
	if gameName == "terraria" && serverCfg.Difficulty == "" {
		serverCfg.Difficulty = "0"
	}
	serverCfg.NodeSelector = parseNodeSelector(opts.Node)
	serverCfg.Gamemode = strings.TrimSpace(opts.Gamemode)
	serverCfg.Seed = strings.TrimSpace(opts.Seed)
	serverCfg.PvP = strings.TrimSpace(opts.PvP)
	serverCfg.ViewDistance = strings.TrimSpace(opts.ViewDistance)
	serverCfg.LevelName = strings.TrimSpace(opts.LevelName)

	limit, err := util.InferMemoryLimit(serverCfg.Memory)
	if err != nil {
		return err
	}
	serverCfg.MemoryLimit = limit

	if gameSupportsAutoBackup(game) {
		serverCfg.AutoBackup = opts.AutoBackup
	} else {
		serverCfg.AutoBackup = false
	}

	if err := validateDeployConfig(serverCfg, state, game); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := game.Validate(serverCfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	manifests, err := game.RenderManifests(serverCfg)
	if err != nil {
		return fmt.Errorf("rendering manifests: %w", err)
	}

	client := k8s.NewClient(cfg.Kubeconfig)
	if err := validateNodeExists(client, serverCfg.NodeSelector); err != nil {
		return err
	}

	if err := client.ApplyAll(manifests); err != nil {
		return fmt.Errorf("applying manifests: %w", err)
	}

	namespace := game.GetNamespace(serverCfg.Name)
	deployment := game.GetDeploymentName(serverCfg.Name)
	if err := client.WaitForReady(namespace, deployment, 180); err != nil {
		fmt.Fprintf(os.Stderr, "warning: server is taking longer than expected to start: %v\n", err)
	}

	state.Add(config.ServerInfo{
		Name:       serverCfg.Name,
		Game:       game.Name(),
		Namespace:  namespace,
		Variant:    serverCfg.Variant,
		AutoBackup: serverCfg.AutoBackup,
		Node:       serverCfg.NodeSelector,
		Port:       serverCfg.Port,
		Memory:     serverCfg.Memory,
		MaxPlayers: serverCfg.MaxPlayers,
		CreatedAt:  time.Now().Format(time.RFC3339),
	})
	if err := config.SaveServerState(cfg, state); err != nil {
		return fmt.Errorf("saving server state: %w", err)
	}

	fmt.Printf("deployed %s (%s)\n", serverCfg.Name, game.Name())
	return nil
}

func RunListNonInteractive(cfg *config.Config, jsonOutput bool) error {
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	client := k8s.NewClient(cfg.Kubeconfig)
	entries := collectServerRows(state, client)

	if jsonOutput {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling list JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No servers deployed.")
		return nil
	}

	fmt.Printf("%-15s %-14s %-8s %-12s %s\n", "NAME", "GAME", "PORT", "NODE", "STATUS")
	for _, entry := range entries {
		fmt.Printf("%-15s %-14s %-8d %-12s %s\n",
			entry.Name,
			entry.Game,
			entry.Port,
			entry.Node,
			entry.Status,
		)
	}

	return nil
}

func RunDeleteNonInteractive(cfg *config.Config, name string, yes bool) error {
	if !yes {
		return fmt.Errorf("deletion requires --yes")
	}

	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	srv, err := findServerByName(state, name)
	if err != nil {
		return err
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	client := k8s.NewClient(cfg.Kubeconfig)
	if err := client.DeleteNamespace(namespace); err != nil {
		return fmt.Errorf("deleting namespace: %w", err)
	}

	state.Remove(srv.Name)
	if err := config.SaveServerState(cfg, state); err != nil {
		return fmt.Errorf("saving server state: %w", err)
	}

	fmt.Printf("deleted %s\n", srv.Name)
	return nil
}

func RunStopNonInteractive(cfg *config.Config, name string) error {
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	srv, err := findServerByName(state, name)
	if err != nil {
		return err
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	client := k8s.NewClient(cfg.Kubeconfig)

	replicas, err := client.GetReplicas(namespace, deployment)
	if err == nil && replicas == 0 {
		fmt.Printf("%s already stopped\n", srv.Name)
		return nil
	}

	if err := client.ScaleDeployment(namespace, deployment, 0); err != nil {
		return fmt.Errorf("stopping server: %w", err)
	}

	fmt.Printf("stopped %s\n", srv.Name)
	return nil
}

func RunStartNonInteractive(cfg *config.Config, name string) error {
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	srv, err := findServerByName(state, name)
	if err != nil {
		return err
	}

	game := games.Get(srv.Game)
	if game == nil {
		return fmt.Errorf("unknown game: %s", srv.Game)
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	client := k8s.NewClient(cfg.Kubeconfig)

	replicas, err := client.GetReplicas(namespace, deployment)
	if err == nil && replicas > 0 {
		fmt.Printf("%s already running\n", srv.Name)
		return nil
	}

	if err := client.ScaleDeployment(namespace, deployment, 1); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	if err := client.WaitForReady(namespace, deployment, 180); err != nil {
		fmt.Fprintf(os.Stderr, "warning: server is taking longer than expected to start: %v\n", err)
	}

	fmt.Printf("started %s\n", srv.Name)
	return nil
}

func RunBackupNonInteractive(cfg *config.Config, name string) error {
	state, err := config.LoadServerState(cfg)
	if err != nil {
		return fmt.Errorf("loading server state: %w", err)
	}

	srv, err := findServerByName(state, name)
	if err != nil {
		return err
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
		fmt.Println("Saving world before backup...")
		if saveErr := client.SaveWorld(namespace, podName, 30); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save world before backup: %v\n", saveErr)
		}
	}

	if err := client.RunBackupJob(namespace, jobName, jobManifest, 120); err != nil {
		return fmt.Errorf("running backup job: %w", err)
	}

	fmt.Printf("backup created: /mnt/das/zplay-backups/%s-%s.tar.gz\n", srv.Name, timestamp)
	return nil
}

func RunStatusNonInteractive(cfg *config.Config, name string) error {
	return RunStatus(cfg, name)
}

func parseNodeSelector(node string) string {
	value := strings.TrimSpace(node)
	if strings.EqualFold(value, "auto") {
		return ""
	}
	return value
}


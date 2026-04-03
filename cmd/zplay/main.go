package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Zyrakk/zplay/internal/cli"
	"github.com/Zyrakk/zplay/internal/config"
)

var Version = "dev"

func main() {
	if len(os.Args) == 1 {
		runInteractive()
		return
	}

	command := os.Args[1]
	if command == "version" {
		fmt.Printf("zplay %s\n", Version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	var actionErr error
	switch command {
	case "deploy":
		actionErr = runDeployCommand(cfg, os.Args[2:])
	case "list":
		actionErr = runListCommand(cfg, os.Args[2:])
	case "delete":
		actionErr = runDeleteCommand(cfg, os.Args[2:])
	case "stop":
		actionErr = runStopCommand(cfg, os.Args[2:])
	case "start":
		actionErr = runStartCommand(cfg, os.Args[2:])
	case "backup":
		actionErr = runBackupCommand(cfg, os.Args[2:])
	case "status":
		actionErr = runStatusCommand(cfg, os.Args[2:])
	case "cleanup":
		actionErr = runCleanupCommand(cfg, os.Args[2:])
	case "upload-world":
		actionErr = runUploadWorldCommand(cfg, os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}

	if actionErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", actionErr)
		os.Exit(1)
	}
}

func runInteractive() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cli.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDeployCommand(cfg *config.Config, args []string) error {
	deployCmd := flag.NewFlagSet("deploy", flag.ContinueOnError)
	deployCmd.SetOutput(os.Stderr)

	gameName := deployCmd.String("game", "", "Game to deploy (terraria, minecraft)")
	variant := deployCmd.String("variant", "vanilla", "Server variant (vanilla, tmodloader, paper, forge)")
	name := deployCmd.String("name", "", "Server name")
	memory := deployCmd.String("memory", "", "Memory request (e.g., 4Gi)")
	node := deployCmd.String("node", "", "Node selector hostname or auto")
	port := deployCmd.Int("port", 0, "Service port")
	password := deployCmd.String("password", "", "Optional server password")
	maxPlayers := deployCmd.Int("max-players", 8, "Max players")
	worldSize := deployCmd.String("world-size", "medium", "World size (small, medium, large)")
	difficulty := deployCmd.String("difficulty", "", "Difficulty level")
	autoBackup := deployCmd.Bool("auto-backup", false, "Enable daily auto-backup")
	gamemode := deployCmd.String("gamemode", "", "Gamemode (survival, creative, adventure, spectator)")
	seed := deployCmd.String("seed", "", "World seed")
	pvp := deployCmd.String("pvp", "", "PvP enabled (true, false)")
	viewDistance := deployCmd.String("view-distance", "", "View distance (2-32)")
	levelName := deployCmd.String("level-name", "", "Level name")

	if err := deployCmd.Parse(args); err != nil {
		return err
	}
	if deployCmd.NArg() != 0 {
		return fmt.Errorf("deploy does not accept positional arguments")
	}

	missing := make([]string, 0, 5)
	if strings.TrimSpace(*gameName) == "" {
		missing = append(missing, "--game")
	}
	if strings.TrimSpace(*name) == "" {
		missing = append(missing, "--name")
	}
	if strings.TrimSpace(*memory) == "" {
		missing = append(missing, "--memory")
	}
	if strings.TrimSpace(*node) == "" {
		missing = append(missing, "--node")
	}
	if *port <= 0 {
		missing = append(missing, "--port")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
	}

	return cli.RunDeployNonInteractive(cfg, cli.DeployOptions{
		Game:         *gameName,
		Variant:      *variant,
		Name:         *name,
		Memory:       *memory,
		Node:         *node,
		Port:         *port,
		Password:     *password,
		MaxPlayers:   *maxPlayers,
		WorldSize:    *worldSize,
		Difficulty:   *difficulty,
		AutoBackup:   *autoBackup,
		Gamemode:     *gamemode,
		Seed:         *seed,
		PvP:          *pvp,
		ViewDistance: *viewDistance,
		LevelName:    *levelName,
	})
}

func runListCommand(cfg *config.Config, args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ContinueOnError)
	listCmd.SetOutput(os.Stderr)
	jsonOutput := listCmd.Bool("json", false, "Output JSON")

	if err := listCmd.Parse(args); err != nil {
		return err
	}
	if listCmd.NArg() != 0 {
		return fmt.Errorf("list does not accept positional arguments")
	}

	return cli.RunListNonInteractive(cfg, *jsonOutput)
}

func runDeleteCommand(cfg *config.Config, args []string) error {
	deleteCmd := flag.NewFlagSet("delete", flag.ContinueOnError)
	deleteCmd.SetOutput(os.Stderr)
	yes := deleteCmd.Bool("yes", false, "Delete without interactive confirmation")

	name, flagArgs, err := splitNameAndFlags("delete", args)
	if err != nil {
		return err
	}
	if err := deleteCmd.Parse(flagArgs); err != nil {
		return err
	}
	if deleteCmd.NArg() != 0 {
		return fmt.Errorf("unexpected delete arguments: %s", strings.Join(deleteCmd.Args(), " "))
	}

	return cli.RunDeleteNonInteractive(cfg, name, *yes)
}

func runStopCommand(cfg *config.Config, args []string) error {
	stopCmd := flag.NewFlagSet("stop", flag.ContinueOnError)
	stopCmd.SetOutput(os.Stderr)
	if err := stopCmd.Parse(args); err != nil {
		return err
	}

	name, err := parseSingleNameArgFromFlagSet("stop", stopCmd)
	if err != nil {
		return err
	}
	return cli.RunStopNonInteractive(cfg, name)
}

func runStartCommand(cfg *config.Config, args []string) error {
	startCmd := flag.NewFlagSet("start", flag.ContinueOnError)
	startCmd.SetOutput(os.Stderr)
	if err := startCmd.Parse(args); err != nil {
		return err
	}

	name, err := parseSingleNameArgFromFlagSet("start", startCmd)
	if err != nil {
		return err
	}
	return cli.RunStartNonInteractive(cfg, name)
}

func runBackupCommand(cfg *config.Config, args []string) error {
	backupCmd := flag.NewFlagSet("backup", flag.ContinueOnError)
	backupCmd.SetOutput(os.Stderr)
	if err := backupCmd.Parse(args); err != nil {
		return err
	}

	name, err := parseSingleNameArgFromFlagSet("backup", backupCmd)
	if err != nil {
		return err
	}
	return cli.RunBackupNonInteractive(cfg, name)
}

func runStatusCommand(cfg *config.Config, args []string) error {
	statusCmd := flag.NewFlagSet("status", flag.ContinueOnError)
	statusCmd.SetOutput(os.Stderr)
	if err := statusCmd.Parse(args); err != nil {
		return err
	}

	name, err := parseSingleNameArgFromFlagSet("status", statusCmd)
	if err != nil {
		return err
	}
	return cli.RunStatusNonInteractive(cfg, name)
}

func runCleanupCommand(cfg *config.Config, args []string) error {
	cleanupCmd := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	cleanupCmd.SetOutput(os.Stderr)
	yes := cleanupCmd.Bool("yes", false, "Delete without interactive confirmation")
	dryRun := cleanupCmd.Bool("dry-run", false, "Preview deletions without executing")
	jsonOutput := cleanupCmd.Bool("json", false, "Output JSON")

	if err := cleanupCmd.Parse(args); err != nil {
		return err
	}
	if cleanupCmd.NArg() != 0 {
		return fmt.Errorf("cleanup does not accept positional arguments")
	}

	return cli.RunCleanupNonInteractive(cfg, cli.CleanupOptions{
		Yes:    *yes,
		DryRun: *dryRun,
		JSON:   *jsonOutput,
	})
}

func parseSingleNameArgFromFlagSet(command string, fs *flag.FlagSet) (string, error) {
	if fs.NArg() != 1 {
		return "", fmt.Errorf("%s requires exactly one server name", command)
	}
	name := strings.TrimSpace(fs.Arg(0))
	if name == "" || strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("%s requires a valid server name", command)
	}
	return name, nil
}

func splitNameAndFlags(command string, args []string) (string, []string, error) {
	name := ""
	flagArgs := make([]string, 0, len(args))

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			continue
		}
		if name == "" {
			name = strings.TrimSpace(arg)
			continue
		}
		return "", nil, fmt.Errorf("%s accepts only one server name", command)
	}

	if name == "" {
		return "", nil, fmt.Errorf("%s requires a server name", command)
	}

	return name, flagArgs, nil
}

func runUploadWorldCommand(cfg *config.Config, args []string) error {
	uploadCmd := flag.NewFlagSet("upload-world", flag.ContinueOnError)
	uploadCmd.SetOutput(os.Stderr)

	path := uploadCmd.String("path", "", "Local path to world directory or archive")
	url := uploadCmd.String("url", "", "URL to download world archive")
	yes := uploadCmd.Bool("yes", false, "Skip confirmation prompt")

	name, flagArgs, err := splitNameAndFlags("upload-world", args)
	if err != nil {
		return err
	}
	if err := uploadCmd.Parse(flagArgs); err != nil {
		return err
	}
	if uploadCmd.NArg() != 0 {
		return fmt.Errorf("unexpected upload-world arguments: %s", strings.Join(uploadCmd.Args(), " "))
	}

	if *path == "" && *url == "" {
		return fmt.Errorf("either --path or --url is required")
	}
	if *path != "" && *url != "" {
		return fmt.Errorf("only one of --path or --url can be specified")
	}

	return cli.RunUploadWorld(cfg, cli.UploadWorldOptions{
		Name: name,
		Path: *path,
		URL:  *url,
		Yes:  *yes,
	})
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  zplay")
	fmt.Fprintln(os.Stderr, "  zplay version")
	fmt.Fprintln(os.Stderr, "  zplay deploy --game <game> --variant <variant> --name <name> --memory <memory> --node <node|auto> --port <port> [--password <password>] [--max-players <n>] [--world-size <size>] [--difficulty <level>] [--gamemode <mode>] [--seed <seed>] [--pvp <bool>] [--view-distance <n>] [--level-name <name>] [--auto-backup]")
	fmt.Fprintln(os.Stderr, "  zplay list [--json]")
	fmt.Fprintln(os.Stderr, "  zplay delete <name> --yes")
	fmt.Fprintln(os.Stderr, "  zplay stop <name>")
	fmt.Fprintln(os.Stderr, "  zplay start <name>")
	fmt.Fprintln(os.Stderr, "  zplay backup <name>")
	fmt.Fprintln(os.Stderr, "  zplay status <name>")
	fmt.Fprintln(os.Stderr, "  zplay cleanup [--yes]")
	fmt.Fprintln(os.Stderr, "  zplay upload-world <name> --path <path> [--yes]")
	fmt.Fprintln(os.Stderr, "  zplay upload-world <name> --url <url> [--yes]")
}

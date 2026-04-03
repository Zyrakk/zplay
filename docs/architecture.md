# Architecture

ZPlay is a Go CLI for deploying and managing game servers on Kubernetes. This document covers the codebase structure, core abstractions, and extension points for contributors.

## Project Structure

```
zplay/
+-- cmd/
|   +-- zplay/
|   |   +-- main.go                       # CLI entry point (interactive + subcommands)
|   +-- zplay-dashboard/
|       +-- main.go                       # Web dashboard HTTP server (client-go)
|       +-- static/                       # Embedded static HTML/JS assets
+-- internal/
|   +-- cli/
|   |   +-- cli.go                        # Interactive TUI menu (Bubble Tea)
|   |   +-- deploy.go                     # Interactive deploy flow + validation
|   |   +-- noninteractive.go             # Non-interactive subcommand handlers
|   |   +-- list.go                       # Server listing + reconciliation
|   |   +-- status.go                     # Detailed server status
|   |   +-- backup.go                     # Interactive backup
|   |   +-- restore.go                    # Interactive restore
|   |   +-- startstop.go                  # Start/stop server
|   |   +-- delete.go                     # Delete server
|   |   +-- console.go                    # Console attach (tmux + fallback)
|   |   +-- logs.go                       # Log viewer
|   +-- config/
|   |   +-- config.go                     # Config loading, server state, reconciliation, file locking
|   +-- games/
|   |   +-- game.go                       # Game interface, registry, ServerConfig, PortToEntrypoint
|   |   +-- render.go                     # Template rendering (RenderTemplates, RenderSingleTemplate)
|   |   +-- terraria/
|   |   |   +-- terraria.go              # Terraria Game implementation
|   |   |   +-- terraria_test.go         # Tests
|   |   |   +-- templates/*.yaml         # 9 K8s manifest templates
|   |   +-- minecraft/
|   |       +-- minecraft.go             # Minecraft Game implementation
|   |       +-- minecraft_test.go        # Tests
|   |       +-- templates/*.yaml         # 9 K8s manifest templates
|   +-- k8s/
|   |   +-- client.go                     # kubectl CLI wrapper
|   +-- util/
|       +-- memory.go                     # Memory parsing utilities
+-- Makefile
+-- Dockerfile.dashboard
+-- VERSION                               # Current version (0.5.0)
+-- go.mod
```

## Game Interface

Every supported game implements the `Game` interface defined in `internal/games/game.go`:

```go
type Game interface {
    Name() string                                          // Identifier: "terraria", "minecraft"
    DisplayName() string                                   // Human-readable: "Terraria", "Minecraft"
    DefaultPort() int                                      // Default listen port for the game
    Validate(cfg *ServerConfig) error                      // Validate config before deploy
    RenderManifests(cfg *ServerConfig) ([]string, error)   // Generate all K8s manifests
    RenderBackupJob(cfg *ServerConfig) (string, error)     // Generate manual backup Job manifest
    RenderRestoreJob(cfg *ServerConfig) (string, error)    // Generate restore Job manifest
    GetDeploymentName(serverName string) string            // Deployment name for a server
    GetNamespace(serverName string) string                 // Namespace for a server
}
```

### Registry Pattern

Games register themselves at import time using Go's `init()` mechanism. The registry lives in `game.go`:

```go
var registry = make(map[string]Game)

func Register(game Game) {
    registry[game.Name()] = game
}
```

Each game package calls `Register` in its `init()` function:

```go
// internal/games/terraria/terraria.go
func init() {
    games.Register(&Terraria{})
}
```

The CLI triggers registration through blank imports in `internal/cli/cli.go`:

```go
import (
    _ "github.com/Zyrakk/zplay/internal/games/minecraft"
    _ "github.com/Zyrakk/zplay/internal/games/terraria"
)
```

At runtime, `games.Available()` returns all registered games. The core CLI code never hardcodes game names -- it discovers them from the registry. This means adding a new game requires zero changes to the CLI's menu, listing, or status logic.

### ServerConfig

`ServerConfig` (in `game.go`) is the shared configuration struct passed to all game operations. It contains common fields (name, port, memory, domain) alongside game-specific fields (e.g., `WorldSize` for Terraria, `ServerType` for Minecraft). Infrastructure defaults (backup paths, storage class, CPU limits) are also carried here so templates can reference them directly.

### PortToEntrypoint

The `PortToEntrypoint` function in `game.go` maps game ports to fixed Traefik entrypoint names:

| Game      | Port  | Entrypoint  |
|-----------|-------|-------------|
| Terraria  | 7777  | terraria1   |
| Terraria  | 7778  | terraria2   |
| Minecraft | 25565 | minecraft1  |
| Minecraft | 25566 | minecraft2  |

Ports outside this map fall back to `zplay-<port>`.

## Template Rendering

Each game embeds its Kubernetes manifest templates using Go's `embed` directive:

```go
//go:embed templates/*.yaml
var templates embed.FS
```

Templates are standard Go `text/template` files that receive a `*ServerConfig` as their data context. The rendering pipeline in `internal/games/render.go` works as follows:

1. **`ApplyInfraDefaults(cfg)`** -- Fills in fallback values for infrastructure fields (storage size, storage class, CPU requests/limits, backup path, backup schedule, backup retention, backup node). This ensures manually constructed configs (e.g., backup/restore flows) have sane defaults even when `NewServerConfig` was not used.

2. **`RenderTemplates(fs, dir, files, cfg)`** -- Iterates over the template file list, parses each one from the embedded filesystem, executes it against the `ServerConfig`, and collects the rendered YAML strings. Empty results are filtered out automatically.

3. **Conditional templates** -- Some templates (e.g., `secret.yaml`) render to an empty string when their conditions are not met (no password configured). The rendering pipeline filters these out, so they never reach `kubectl apply`.

4. **`RenderSingleTemplate(fs, dir, file, cfg)`** -- Used for one-off renders like backup and restore job manifests.

### Template Set

Each game provides 9 templates:

| Template             | Purpose                                    |
|----------------------|--------------------------------------------|
| `namespace.yaml`     | Dedicated namespace for the server         |
| `volume.yaml`        | PersistentVolumeClaim for world data       |
| `secret.yaml`        | Server password (conditional)              |
| `deployment.yaml`    | Game server Deployment                     |
| `service.yaml`       | ClusterIP or NodePort Service              |
| `ingress.yaml`       | Traefik IngressRouteTCP for external access|
| `backup-job.yaml`    | One-shot manual backup Job                 |
| `cronjob-backup.yaml`| Scheduled automatic backup CronJob         |
| `restore-job.yaml`   | Restore from backup Job                    |

## Kubernetes Interaction

The CLI does **not** use `client-go` for cluster operations. Instead, `internal/k8s/client.go` wraps the `kubectl` binary, executing it as a subprocess with the configured `KUBECONFIG`. This keeps the CLI binary small and avoids compiling the full Kubernetes client library into it.

The wrapper also supports `zcloud k` as an alternative transport when the kubeconfig path contains `.zcloud` and the `zcloud` binary is available.

### Operations by Category

**Deploy and Lifecycle**
- `Apply` / `ApplyAll` -- Apply one or more YAML manifests via stdin
- `DeleteNamespace` -- Remove a server's entire namespace
- `ScaleDeployment` -- Scale replicas (start/stop)
- `WaitForReady` -- Block until a Deployment is available
- `NamespaceExists` -- Check if a namespace already exists

**Status and Inspection**
- `GetPodStatus` -- Pod phase (Running, Pending, etc.)
- `GetPodName` -- Pod name by label selector
- `GetPodNodeName` -- Node where the pod is scheduled
- `GetPodStartTime` -- Pod start timestamp
- `GetPodTop` -- Live CPU and memory usage from metrics-server
- `GetReplicas` -- Current replica count
- `GetDeploymentResources` -- Memory/CPU requests and limits
- `GetPVCInfo` -- Storage request and storage class
- `GetServicePort` -- Exposed service port
- `GetNodes` -- List cluster nodes
- `IsConnected` -- Verify cluster connectivity

**Discovery**
- `DiscoverServers` -- Find all zplay-managed deployments across namespaces (by `app=zplay` label)
- `GetDeployments` -- List deployments matching a label selector

**Console and Logs**
- `AttachConsole` -- Interactive attach to a deployment (stdin/stdout)
- `AttachConsoleViaTmux` -- Attach via a tmux session for better terminal handling
- `Logs` -- Stream or tail deployment logs
- `Exec` / `ExecNoTTY` -- Execute commands inside a pod

**Backup and Restore**
- `RunJob` -- Apply a Job manifest and wait for completion
- `RunBackupJob` -- Run a backup job (wraps RunJob)
- `RunJobAndGetLogs` -- Run a job and capture its output
- `SaveWorld` -- Send a save command to the game server via tmux
- `HasBackupCronJob` -- Check if automatic backups are configured
- `GetLastBackupTimestamp` -- Timestamp of the most recent backup job

**Cleanup**
- `GetReleasedPVs` -- Find PersistentVolumes in Released state from zplay namespaces
- `DeletePV` -- Remove a released PersistentVolume

## State Management

ZPlay maintains local state in `~/.zplay/servers.yaml`, tracking deployed servers alongside metadata (game, port, memory, node, creation time). The config and infrastructure settings live in `~/.zplay/config.yaml`.

### File Locking

All reads and writes to `servers.yaml` are wrapped in `withStateLock`, which uses `syscall.Flock` (POSIX advisory locking) on a `.lock` file adjacent to the state file. This prevents corruption if multiple zplay instances run concurrently.

### Reconciliation

Local state can drift from cluster reality (e.g., a server was deleted via kubectl directly, or deployed from another machine). The `Reconcile` function compares local state against cluster discovery:

1. `DiscoverServers()` queries the cluster for all deployments labeled `app=zplay`.
2. `Reconcile(state, discovered)` returns two lists:
   - **Added** -- Servers found in the cluster but missing from local state.
   - **Orphaned** -- Servers in local state but not found in the cluster.
3. The CLI uses these lists to prompt the user for state cleanup or adoption.

State sync also updates runtime fields (port, memory) from the cluster for tracked servers, so local state stays accurate even if a server's resources were modified externally.

## Dashboard

The web dashboard is a separate binary (`cmd/zplay-dashboard/`) that provides a browser-based view of server status.

Key differences from the CLI:

- **Uses `client-go`** for direct Kubernetes API access (not the kubectl wrapper). This is the only component that imports `k8s.io/client-go` and `k8s.io/metrics`.
- Serves embedded static HTML/JS files (`//go:embed static`).
- Exposes a REST API: `/api/servers` (server list with status) and `/api/health` (health check).
- Runs as a container: `ghcr.io/zyrakk/zplay-dashboard:latest`.
- Multi-architecture: builds for `linux/arm64` and `linux/amd64`.
- Configured via `Dockerfile.dashboard` at the repository root.

## Adding a New Game

To add support for a new game, follow these steps:

### 1. Create the game directory

```
internal/games/<name>/
internal/games/<name>/templates/
```

### 2. Implement the Game interface

Create `internal/games/<name>/<name>.go`:

```go
package <name>

import (
    "embed"

    "github.com/Zyrakk/zplay/internal/games"
)

//go:embed templates/*.yaml
var templates embed.FS

func init() {
    games.Register(&YourGame{})
}

type YourGame struct{}

func (g *YourGame) Name() string        { return "<name>" }
func (g *YourGame) DisplayName() string { return "<Display Name>" }
func (g *YourGame) DefaultPort() int    { return <port> }

func (g *YourGame) Validate(cfg *games.ServerConfig) error {
    // Validate game-specific config fields
}

func (g *YourGame) RenderManifests(cfg *games.ServerConfig) ([]string, error) {
    games.ApplyInfraDefaults(cfg)
    return games.RenderTemplates(templates, "templates", []string{
        "namespace.yaml", "volume.yaml", "secret.yaml",
        "deployment.yaml", "service.yaml", "ingress.yaml",
        "cronjob-backup.yaml",
    }, cfg)
}

func (g *YourGame) RenderBackupJob(cfg *games.ServerConfig) (string, error) {
    games.ApplyInfraDefaults(cfg)
    return games.RenderSingleTemplate(templates, "templates", "backup-job.yaml", cfg)
}

func (g *YourGame) RenderRestoreJob(cfg *games.ServerConfig) (string, error) {
    games.ApplyInfraDefaults(cfg)
    return games.RenderSingleTemplate(templates, "templates", "restore-job.yaml", cfg)
}

func (g *YourGame) GetDeploymentName(serverName string) string {
    return serverName + "-yourgame"
}

func (g *YourGame) GetNamespace(serverName string) string {
    return "zplay-" + serverName
}
```

### 3. Create the 9 Kubernetes manifest templates

Place these in `internal/games/<name>/templates/`:

| File                  | Required fields from ServerConfig                       |
|-----------------------|---------------------------------------------------------|
| `namespace.yaml`      | Name                                                    |
| `volume.yaml`         | Name, StorageSize, StorageClass                         |
| `secret.yaml`         | Password (render empty if no password)                  |
| `deployment.yaml`     | Name, Memory, MemoryLimit, CPURequest, CPULimit, Port   |
| `service.yaml`        | Name, Port                                              |
| `ingress.yaml`        | Name, Port, Entrypoint, Domain                          |
| `backup-job.yaml`     | Name, Timestamp, BackupPath, BackupNode                 |
| `cronjob-backup.yaml` | Name, BackupPath, BackupSchedule, BackupRetention       |
| `restore-job.yaml`    | Name, BackupFile, BackupPath, BackupNode                |

Use existing game templates (e.g., `internal/games/terraria/templates/`) as reference.

### 4. Register via blank import

Add the blank import to `internal/cli/cli.go`:

```go
import (
    _ "github.com/Zyrakk/zplay/internal/games/<name>"
)
```

This triggers the `init()` function at startup and registers the game in the global registry.

### 5. Add port-to-entrypoint mappings

In `internal/games/game.go`, add entries to the `PortToEntrypoint` function:

```go
"<name>": {
    <port1>: "<name>1",
    <port2>: "<name>2",
},
```

### 6. Add game-specific deploy prompts

In `internal/cli/deploy.go`, add a case for your game in the interactive deploy flow. This is where you prompt the user for game-specific options (variant, world settings, version, etc.) that populate the `ServerConfig` fields your templates depend on.

### 7. Add allowed ports

In `internal/cli/deploy.go`, add a case to the `allowedEntrypointPorts` function:

```go
case "<name>":
    return []int{<port1>, <port2>}
```

This ensures deploy validation rejects ports that have no configured Traefik entrypoint.

### 8. Configure Traefik entrypoints

On the cluster, add TCP entrypoints to the Traefik configuration for each port your game uses. Each entrypoint name must match the values in `PortToEntrypoint`. Without this, ingress routing will not work even if all code changes are correct.

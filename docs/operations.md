# Operations Guide

This document covers the operational workflows available in ZPlay for managing the lifecycle of game servers: backup and restore, start and stop, monitoring, console access, cleanup, and state reconciliation.

## Backup

ZPlay supports both manual on-demand backups and automated scheduled backups. All backups are stored as compressed archives at `/mnt/das/zplay-backups/` on the configured backup node.

### Manual Backup

```bash
zplay backup <name>
```

This command performs the following steps:

1. Sends a save command to the running server pod to flush the world state to disk.
2. Creates a Kubernetes Job on the backup node that archives the server's `/data` directory.
3. Produces a file named `<name>-<timestamp>.tar.gz` in the backup directory.

The backup job runs as an `alpine:3.19` container using a hostPath volume mounted at `/mnt/das/zplay-backups`.

In interactive mode, the same operation is available under the "Backup server" menu option, which presents a list of deployed servers to choose from.

### Auto-Backup

When auto-backup is enabled for a server (via `--auto-backup` during deploy, or the interactive prompt), ZPlay creates a Kubernetes CronJob that runs on a configured schedule.

- **Default schedule:** `0 4 * * *` (4:00 AM daily)
- **Retention:** Keeps the last N backups per server (default 7). Older backups are automatically deleted by the cronjob script.
- **Node:** Backup jobs run on the configured backup node (default `oracle1`).

The schedule and retention values can be changed in `config.yaml`. See the [Configuration Reference](configuration.md) for details.

### Backup Location

All backup files are stored at:

```
/mnt/das/zplay-backups/<server-name>-<timestamp>.tar.gz
```

This path corresponds to a hostPath volume on the backup node. Ensure the directory exists before running backups:

```bash
mkdir -p /mnt/das/zplay-backups
```

## Restore

Restore is currently available only through the interactive menu. There is no CLI subcommand for restore yet.

### Interactive Flow

1. Launch `zplay` and select "Restore backup".
2. Select the server to restore.
3. ZPlay lists available backup files for that server by running a temporary Kubernetes Job that reads the backup directory.
4. Select the backup file to restore from.
5. Type the server name to confirm the operation.

### Execution Steps

Once confirmed, the restore proceeds as follows:

1. **Stop** the server by scaling the deployment to 0 replicas.
2. **Run a restore job** that clears the contents of `/data` and extracts the selected `.tar.gz` archive into it.
3. **Start** the server by scaling the deployment back to 1 replica.
4. **Wait** for the server to reach a ready state.

**Warning:** Restore overwrites all current server data. The existing world, configuration, and any unsaved state in `/data` will be permanently replaced by the backup contents. There is no undo.

## Start and Stop

### Stop a Server

```bash
zplay stop <name>
```

Scales the server's Kubernetes Deployment to 0 replicas. The Persistent Volume Claim (PVC) is preserved -- no data is lost. The server simply stops running.

### Start a Server

```bash
zplay start <name>
```

Scales the server's Deployment back to 1 replica and waits for the pod to reach a ready state (timeout: 180 seconds).

### Interactive Mode

In the interactive menu, "Start/Stop server" displays all servers with their current status (Running or Stopped), allowing you to toggle them directly.

### Idempotent Behavior

Both commands are idempotent:

- Stopping an already-stopped server prints "already stopped" and exits cleanly.
- Starting an already-running server prints "already running" and exits cleanly.

## Server Status

```bash
zplay status <name>
```

Displays detailed information about a server, organized into the following categories:

### Server Metadata

- Name, game (with variant), creation date
- Address (domain and port, e.g., `play.zyrak.cloud:7777`)

### Runtime

- Status: Running, Stopped, or Unknown
- Node the pod is scheduled on
- Uptime since the pod started

### Resources

- Memory: current usage, request, and limit
- CPU: current usage and limit
- Resource metrics require `metrics-server` to be installed in the cluster. If unavailable, these fields display "N/A" instead of failing.

### Storage

- PVC size and StorageClass

### Backup

- Auto-backup enabled or disabled
- Timestamp of the last backup (if any)

The status command is designed for graceful degradation. Missing or unavailable data points are shown as N/A rather than causing errors.

## Server Console

The console feature attaches your terminal to a running server's stdin and stdout, allowing you to issue in-game commands directly.

### Interactive Flow

1. Launch `zplay` and select "Server console".
2. Select the server to connect to.

### With tmux (Recommended)

If `tmux` is detected on your system (via `exec.LookPath`), ZPlay wraps the `kubectl attach` session inside a tmux session. This provides safe detachment:

- **Detach:** Press `Ctrl+B`, then `D`. The server continues running in the background.
- **Reattach:** Run the console command again, or use `tmux attach`.

### Without tmux

If tmux is not available, ZPlay performs a direct `kubectl attach`. In this mode:

- **Ctrl+C will stop the server.** There is no safe way to detach without tmux.

### Fallback

If the attach operation fails (for example, if the pod is not in a ready state), ZPlay automatically falls back to streaming live deployment logs instead.

## Viewing Logs

### Interactive Flow

1. Launch `zplay` and select "View logs".
2. Select the server.
3. Choose whether to follow the log stream (Y/n).

ZPlay streams the Kubernetes Deployment logs to your terminal. Press `Ctrl+C` to stop following and return to the menu.

## Cleanup

```bash
zplay cleanup [--yes] [--dry-run] [--json]
```

Discovers orphaned ZPlay namespaces in the cluster -- namespaces that exist in Kubernetes but are not tracked in the local `servers.yaml` state file.

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview what would be deleted without making changes |
| `--json` | Output results as structured JSON |
| `--yes` | Skip the confirmation prompt and proceed immediately |

### Examples

Preview orphaned resources:

```bash
zplay cleanup --dry-run
```

Remove orphaned resources without confirmation:

```bash
zplay cleanup --yes
```

In interactive mode, this operation is available as "Cleanup resources" in the main menu.

## Reconciliation

ZPlay maintains local state in `~/.zplay/servers.yaml` that can drift from the actual cluster state. The reconciliation process detects and resolves these discrepancies.

### When It Runs

Reconciliation runs automatically when you execute `zplay list` in interactive mode. A non-interactive `--sync` flag is planned but not yet implemented.

### What It Checks

ZPlay discovers all Deployments in the cluster with the label `app=zplay` and compares them against the local `servers.yaml` file.

### Discrepancy Handling

**Servers found in the cluster but missing locally (added):**
ZPlay offers to adopt these servers into your local state, bringing them under management without redeploying.

**Servers present locally but missing from the cluster (orphaned):**
ZPlay offers to clean these stale entries from your local state file.

### State Sync

For servers that are tracked in both the local state and the cluster, reconciliation also syncs runtime values (port, memory) from the actual cluster Deployment back into `servers.yaml`, ensuring local state reflects reality.

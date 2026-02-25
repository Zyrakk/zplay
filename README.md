# ZPlay

ZPlay is a CLI to deploy and operate game servers on Kubernetes (k3s).
Current CLI version: `0.5.0`.

## Project Status

Current status (February 25, 2026):

- Phase 0 - Foundation: completed.
- Phase 1 - Robustness: completed.
- Phase 2 - Terraria variants (including tModLoader): completed.
- Phase 3 - Backups and restore: completed.
- Phase 4 - Usability: completed.
  - Cluster reconciliation on `list` (adopt/cleanup local state).
  - Direct subcommands with flags (`deploy`, `list`, `delete`, `start`, `stop`, `backup`, `status`).
  - Detailed per-server status view.
- Post-Phase 4 hardening (February 25, 2026):
  - Terminal reset after Bubble Tea (`stty sane`) before executing actions.
  - Terraria console attach via local `tmux` with clean detach (`Ctrl+B`, `D`) when available.
  - `View logs` follow mode exits cleanly on `Ctrl+C`.
  - Terraria deployment template uses `exec` probes (no TCP probe connection spam in logs on new deploys).
- Phase 5 (Minecraft): planned in `docs/roadmap.md`.

## Supported Games

- Terraria (vanilla): stable.
- Terraria (tModLoader): supported (requires x86 node `lake`).
- Minecraft: not implemented yet.

## Requirements

- Go 1.22+
- `kubectl` configured against your cluster
- Access to your Kubernetes cluster (`zcloud login` or direct kubeconfig)
- Traefik with TCP entrypoints for game ports
- `tmux` (optional, recommended for safe detach from `Server console`)

## Installation

### Option A: Build and run locally

```bash
git clone https://github.com/Zyrakk/zplay.git
cd zplay
make deps
make dev
```

### Option B: Install system-wide (recommended)

```bash
git clone https://github.com/Zyrakk/zplay.git
cd zplay
make deps
make build
sudo cp dist/zplay /usr/local/bin/zplay
```

Equivalent shortcut:

```bash
make install
```

### Option C: Install for current user (no sudo)

```bash
git clone https://github.com/Zyrakk/zplay.git
cd zplay
make deps
make build
mkdir -p "$HOME/.local/bin"
cp dist/zplay "$HOME/.local/bin/zplay"
export PATH="$HOME/.local/bin:$PATH"
```

## Cluster Login and First Run

If you use ZCloud:

```bash
zcloud login
kubectl --kubeconfig "$HOME/.zcloud/kubeconfig" get nodes
zplay
```

By default, ZPlay resolves kubeconfig in this order:
1. `kubeconfig` in `~/.zplay/config.yaml` (if explicitly set and not the legacy default)
2. `$KUBECONFIG`
3. `~/.kube/config`
4. `~/.zcloud/kubeconfig` (legacy fallback)

## Usage Modes

### Interactive mode (default)

```bash
zplay
```

Main menu:

```text
Deploy server
List servers
Start/Stop server
Server status
Backup server
Restore backup
Delete server
Server console
View logs
Exit
```

### Direct mode (subcommands)

```bash
zplay version
zplay deploy --game terraria --variant vanilla --name myserver --memory 4Gi --node oracle1 --port 7777 [--password xxx] [--max-players 8] [--world-size medium] [--difficulty 0] [--auto-backup]
zplay list [--json]
zplay delete <name> --yes
zplay stop <name>
zplay start <name>
zplay backup <name>
zplay status <name>
```

## Reconciliation

`List servers` (interactive) reconciles local state (`~/.zplay/servers.yaml`) with the cluster by discovering deployments labeled `app=zplay`.

- Servers found in cluster but missing locally can be adopted.
- Servers present locally but missing in cluster can be cleaned.

TODO: non-interactive sync mode (`zplay list --sync`) is not implemented yet.

## Detailed Status

`zplay status <name>` (or interactive `Server status`) shows:

- Server metadata (game, variant, created date, public address)
- Runtime state (status, node, uptime)
- Resources (CPU/memory usage when metrics-server is available, request/limit)
- Storage info (PVC)
- Backup info (auto-backup enabled/disabled, last backup timestamp)

Unavailable metrics are displayed as `N/A` without failing the command.

## Traefik EntryPoints

ZPlay uses fixed game entrypoint mapping in code:

- Terraria `7777` -> `terraria1`
- Terraria `7778` -> `terraria2`
- Minecraft `25565` -> `minecraft1` (future)
- Minecraft `25566` -> `minecraft2` (future)

Make sure these TCP entrypoints exist in your Traefik configuration.

## Configuration

`~/.zplay/config.yaml`:

```yaml
domain: play.zyrak.cloud
kubeconfig: ~/.zcloud/kubeconfig
node_selector: ""
data_path: ~/.zplay
```

Server state is stored in `~/.zplay/servers.yaml`.

Example:

```yaml
servers:
  - name: vanilla
    game: terraria
    namespace: zplay-vanilla
    variant: vanilla
    auto_backup: true
    node: oracle1
    port: 7777
    memory: 4Gi
    max_players: 8
    created_at: "2025-01-15T10:30:00Z"
```

## Development

```bash
make dev
make test
make build-all
make lint
make fmt
```

## License

MIT

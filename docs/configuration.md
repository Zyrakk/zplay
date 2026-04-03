# Configuration Reference

ZPlay v0.5.0 uses two YAML files for persistent state: a user-editable configuration file and a managed server state file. Both are stored under the data directory (default `~/.zplay/`).

## Config File

**Location:** `~/.zplay/config.yaml`

The configuration file is created automatically on first run with sensible defaults. Edit it directly with any text editor. Changes take effect on the next command invocation.

File permissions are set to `0600` (owner read/write only).

## Complete Example

Below is the full configuration with all default values:

```yaml
domain: play.zyrak.cloud
kubeconfig: ""  # resolved automatically; see Kubeconfig Resolution below
node_selector: ""
data_path: ~/.zplay
backup:
  path: /mnt/das/zplay-backups
  schedule: "0 4 * * *"
  retention: 7
  node: oracle1
storage:
  size: 10Gi
  class: nfs-shared
defaults:
  memory_request: 4Gi
  memory_limit: 8Gi
  cpu_request: 500m
  cpu_limit: "2"
probes:
  vanilla_initial_delay: 120
  tmodloader_initial_delay: 300
```

## Field Reference

### Top-Level Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `domain` | string | `play.zyrak.cloud` | Base domain used for ingress routing to game servers. |
| `kubeconfig` | string | *(resolved via fallback chain)* | Path to the kubeconfig file. When empty or unset, resolved automatically (see Kubeconfig Resolution below). |
| `node_selector` | string | `""` | Kubernetes node selector expression applied to all server deployments. Leave empty for no constraint. |
| `data_path` | string | `~/.zplay` | Directory where ZPlay stores its data files, including `servers.yaml`. |

### Backup Fields (`backup.*`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `backup.path` | string | `/mnt/das/zplay-backups` | Host path where backup archives are stored. |
| `backup.schedule` | string | `0 4 * * *` | Cron expression for automated backup scheduling (default: daily at 04:00). |
| `backup.retention` | int | `7` | Number of backup copies to retain before older ones are pruned. |
| `backup.node` | string | `oracle1` | Kubernetes node where backup jobs are scheduled (must have access to the backup path). |

### Storage Fields (`storage.*`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `storage.size` | string | `10Gi` | Size of the PersistentVolumeClaim provisioned for each game server. |
| `storage.class` | string | `nfs-shared` | Kubernetes StorageClass used for PVC provisioning. |

### Resource Defaults (`defaults.*`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `defaults.memory_request` | string | `4Gi` | Kubernetes memory request applied to server containers. |
| `defaults.memory_limit` | string | `8Gi` | Kubernetes memory limit applied to server containers. |
| `defaults.cpu_request` | string | `500m` | Kubernetes CPU request applied to server containers. |
| `defaults.cpu_limit` | string | `2` | Kubernetes CPU limit applied to server containers. |

### Probe Timeouts (`probes.*`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `probes.vanilla_initial_delay` | int | `120` | Initial delay in seconds before liveness/readiness probes begin for vanilla game servers. |
| `probes.tmodloader_initial_delay` | int | `300` | Initial delay in seconds before liveness/readiness probes begin for tModLoader servers (longer due to mod loading). |

## Kubeconfig Resolution

When `kubeconfig` is not explicitly set in the configuration file, ZPlay resolves it using the following fallback chain:

1. **Config file value** -- If `kubeconfig` is set in `config.yaml` to an explicit, non-legacy path, that value is used directly (even if the file does not yet exist).
2. **`$KUBECONFIG` environment variable** -- If the environment variable is set and non-empty, its value is used. Colon-separated multi-path values are supported.
3. **`~/.kube/config`** -- The standard kubectl default location is checked. Used if the file exists.
4. **`~/.zcloud/kubeconfig`** -- Legacy fallback path from earlier tooling. Used if the file exists, or as the final default when no other path resolves.

Tilde (`~`) expansion and environment variable substitution are applied at every step.

## Server State File

**Location:** `~/.zplay/servers.yaml`

The server state file is a managed file that tracks all game servers deployed through ZPlay. It is not intended for manual editing. ZPlay reads and writes this file automatically during server lifecycle operations (create, delete, list).

File permissions are set to `0600`. Concurrent access is protected by an exclusive file lock (see State Reconciliation below).

## Server State Format

Each entry in the `servers` list records the deployment metadata for a single game server:

```yaml
servers:
  - name: terraria-survival
    game: terraria
    namespace: zplay
    variant: vanilla
    auto_backup: true
    node: oracle1
    port: 7777
    memory: 4Gi
    max_players: 16
    created_at: "2026-03-15T10:30:00Z"
  - name: minecraft-creative
    game: minecraft
    namespace: zplay
    variant: paper
    auto_backup: false
    node: oracle1
    port: 25565
    memory: 8Gi
    max_players: 20
    created_at: "2026-03-20T14:00:00Z"
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique server name, used as the Kubernetes deployment and service name. |
| `game` | string | Game type identifier (e.g., `terraria`, `minecraft`). |
| `namespace` | string | Kubernetes namespace where the server is deployed. Omitted when empty. |
| `variant` | string | Game variant (e.g., `vanilla`, `tmodloader`, `paper`, `forge`). Omitted when empty. |
| `auto_backup` | bool | Whether automated backups are enabled for this server. Omitted when false. |
| `node` | string | Kubernetes node the server is scheduled on. |
| `port` | int | Host port exposed for the game server. |
| `memory` | string | Memory allocation for the server container. |
| `max_players` | int | Maximum player count configured for the game server. |
| `created_at` | string | ISO 8601 timestamp recording when the server was first deployed. |

## State Reconciliation

When running the interactive `list` command, ZPlay reconciles local state against the live cluster. It queries Kubernetes for all deployments labeled `app=zplay` and compares them to entries in `servers.yaml`:

- **Untracked servers** (present in the cluster but missing locally) are offered for adoption into the state file.
- **Orphaned entries** (present locally but missing from the cluster) are offered for cleanup from the state file.

All reads and writes to `servers.yaml` are protected by an exclusive file lock (`flock`) via a `.lock` sidecar file. This prevents data corruption when multiple ZPlay processes access the state file concurrently.

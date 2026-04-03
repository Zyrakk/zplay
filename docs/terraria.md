# Terraria Server Guide

ZPlay supports deploying Terraria servers in two variants: vanilla and tModLoader. Both variants can be deployed through the interactive TUI or directly via CLI flags.

## Variants

| Variant | Image | Node Constraint | Recommended Memory |
|---------|-------|-----------------|-------------------|
| vanilla | `hexlo/terraria-server-docker:latest` | Any node | 4Gi |
| tModLoader | `jacobsmile/tmodloader1.4:latest` | `lake` only (x86 required) | 4Gi+ (8Gi for large mod packs like Calamity) |

tModLoader runs under Mono/dotnet and requires an x86 architecture node. The only x86 node in the cluster is `lake`. Attempting to deploy tModLoader on any other node will fail validation.

## Server Parameters

| Parameter | CLI Flag | Interactive Prompt | Default | Validation |
|-----------|----------|--------------------|---------|------------|
| Server name | `--name` | Text input | Required | 2-20 chars, lowercase alphanumeric + hyphens, must start and end with a letter/digit. Regex: `^[a-z][a-z0-9-]{0,18}[a-z0-9]$` |
| Variant | `--variant` | Menu (1=vanilla, 2=tmodloader) | `vanilla` | Must be `vanilla` or `tmodloader` |
| Memory | `--memory` | Text input with current default shown | `4Gi` | Must match `^\d+[GM]i$`. Raspberry node capped at 4Gi |
| Node | `--node` | Menu (oracle1/oracle2/raspberry/auto) | `auto` (interactive), required (CLI) | tModLoader forces `lake`. Raspberry limits memory to 4Gi |
| Port | `--port` | Text input with auto-increment suggestion | `7777` | Must be 7777 or 7778 |
| Max players | `--max-players` | Text input | `8` | 1-255 |
| World size | `--world-size` | Menu (small/medium/large) | `medium` | Maps to 1/2/3 internally |
| Difficulty | `--difficulty` | Menu (Classic/Expert/Master/Journey) | `0` (Classic) | Must be 0, 1, 2, or 3 |
| Password | `--password` | Text input (optional) | None | Stored as a Kubernetes Secret |
| Auto-backup | `--auto-backup` | Y/n prompt | Enabled (interactive), disabled (CLI) | Boolean flag |

In CLI mode, `--game`, `--name`, `--memory`, `--node`, and `--port` are all required flags. Omitting any of them produces an error.

## Deploy Examples

### Interactive Mode

Run `zplay` with no arguments to enter the interactive TUI. The deploy flow walks through each parameter in sequence:

1. Select game (Terraria)
2. Enter server name
3. Choose variant (vanilla or tModLoader)
4. Select target node
5. Set memory allocation
6. Choose world size, difficulty, max players
7. Set password (optional)
8. Choose port
9. Enable or disable daily auto-backup
10. Review summary and confirm

### CLI: Vanilla Server

```bash
zplay deploy --game terraria --variant vanilla --name survival \
  --memory 4Gi --node oracle1 --port 7777 \
  --world-size large --difficulty 0 --max-players 16 --auto-backup
```

### CLI: tModLoader Server

```bash
zplay deploy --game terraria --variant tmodloader --name modded \
  --memory 4Gi --node lake --port 7778 \
  --world-size medium --difficulty 1 --max-players 8 \
  --password "secretpass" --auto-backup
```

When deploying tModLoader, the `--node` flag must be set to `lake`. Any other value will fail validation because tModLoader requires x86 architecture.

## tModLoader Mod Installation

After deploying a tModLoader server, there are two methods to install mods.

### Method 1: Workshop IDs (Recommended)

Set the `TMOD_AUTODOWNLOAD` and `TMOD_ENABLEDMODS` environment variables on the deployment using comma-separated Steam Workshop IDs. The pod will restart automatically and download the mods on startup.

```bash
kubectl set env deployment/modded-terraria -n zplay-modded \
  TMOD_AUTODOWNLOAD="2824688072,2824688804" \
  TMOD_ENABLEDMODS="2824688072,2824688804"
```

Replace `modded` with your actual server name. Both variables must contain the same list of Workshop IDs.

### Method 2: Manual Copy

Copy `.tmod` files directly into the container's mod directory, then restart the server.

```bash
# Copy the mod file into the running pod
kubectl cp ./CalamityMod.tmod zplay-modded/<pod-name>:/data/tModLoader/Mods/

# Restart the server to load the new mod
zplay stop modded && zplay start modded
```

To find the pod name:

```bash
kubectl get pods -n zplay-modded
```

### Mod Persistence

Mod data is stored under `/data/` inside the container, which is backed by a PersistentVolumeClaim. Mods persist across pod restarts, redeployments, and stop/start cycles.

## Data Paths

| Variant | World Data Path |
|---------|----------------|
| vanilla | `/root/.local/share/Terraria/Worlds` |
| tModLoader | `/data` |

Both paths are mounted from a PVC (`<server-name>-pvc`) using the `nfs-shared` storage class with a default size of 10Gi.

## Networking

Terraria servers use TCP game traffic on port 7777 (inside the container). External access is routed through Traefik using an `IngressRouteTCP` resource with `HostSNI(*)` matching.

| Property | Value |
|----------|-------|
| Container port | 7777 (TCP) |
| Allowed external ports | 7777, 7778 |
| Ingress type | Traefik `IngressRouteTCP` |
| Connection address | `play.zyrak.cloud:<port>` |

Each server gets a dedicated Traefik entrypoint mapped to its external port. A maximum of two concurrent Terraria servers can run because only ports 7777 and 7778 have configured entrypoints.

To connect from the Terraria client, use:

```
play.zyrak.cloud:7777
```

## Health Probes

Both liveness and readiness probes use the same exec check:

```bash
pgrep -f 'TerrariaServer|mono|dotnet|tModLoader' || exit 1
```

### Probe Timing by Variant

| Probe | Vanilla | tModLoader |
|-------|---------|------------|
| Liveness initial delay | 120s | 300s |
| Readiness initial delay | 60s | 180s |
| Liveness period | 30s | 30s |
| Readiness period | 10s | 15s |
| Failure threshold | 3 | 5 |

### Custom Probe Delays

Override the initial delay values in `~/.zplay/config.yaml`:

```yaml
probes:
  vanilla_initial_delay: 120
  tmodloader_initial_delay: 300
```

The readiness delay is derived automatically: 1/2 of the liveness delay for vanilla, 3/5 for tModLoader.

## Resource Defaults

| Resource | Default |
|----------|---------|
| Memory request | 4Gi |
| Memory limit | 8Gi (2x request) |
| CPU request | 500m |
| CPU limit | 2 |
| Storage size | 10Gi |
| Storage class | nfs-shared |

When memory is changed during deployment, the memory limit is automatically set to 2x the request value. For example, setting `--memory 6Gi` results in a 12Gi limit.

These defaults can be overridden in `~/.zplay/config.yaml`:

```yaml
defaults:
  memory_request: "4Gi"
  memory_limit: "8Gi"
  cpu_request: "500m"
  cpu_limit: "2"
storage:
  size: "10Gi"
  class: "nfs-shared"
```

## Known Limitations

- **No custom world upload via CLI.** There is no built-in command to upload an existing `.wld` file. Use `kubectl cp` to manually copy world files into the container's data path.
- **No world seed configuration.** The server generates a random world on first start. Seed selection is not exposed as a parameter.
- **No TShock support.** Only the vanilla and tModLoader server images are supported. TShock (modded vanilla with plugin API) is not available.
- **tModLoader requires x86.** The tModLoader image only runs on x86 architecture. The sole compatible node in the cluster is `lake`.
- **Maximum 2 concurrent Terraria servers.** Only ports 7777 and 7778 have configured Traefik entrypoints, limiting the cluster to two simultaneous Terraria instances.

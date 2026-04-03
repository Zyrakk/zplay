# Minecraft Server Guide

ZPlay deploys Minecraft servers on Kubernetes using the [itzg/minecraft-server](https://hub.docker.com/r/itzg/minecraft-server) Docker image (`itzg/minecraft-server:latest`). Three server variants are supported, all sharing the same image with different `TYPE` configurations.

## Variants

| Variant | TYPE env var | Description | Use case |
|---------|-------------|-------------|----------|
| vanilla | `VANILLA` | Unmodified Minecraft server from Mojang | Standard survival/creative, no plugins or mods needed |
| paper | `PAPER` | Performance-optimized Paper fork | Better tick rates, plugin support, larger player counts |
| forge | `FORGE` | Modded server with Forge mod loader | Custom mods, modpacks, heavily modified gameplay |

The default variant is `vanilla` when not specified.

## Server Parameters

| Parameter | CLI Flag | Interactive | Default | Validation |
|-----------|----------|-------------|---------|------------|
| Server name | `--name` | Prompted | Required | 2-20 characters, RFC 1123 compliant |
| Variant | `--variant` | Menu (1=vanilla, 2=paper, 3=forge) | `vanilla` | Must be `vanilla`, `paper`, or `forge` |
| Memory | `--memory` | Prompted | `4Gi` | Must match `^\d+[GM]i$` (e.g. `4Gi`, `512Mi`) |
| Node | `--node` | Menu (oracle1/oracle2/raspberry/auto) | `auto` | `raspberry` enforces max `4Gi` memory |
| Port | `--port` | Prompted with auto-increment | `25565` | Only `25565` or `25566` allowed |
| Max players | `--max-players` | Prompted | `8` | Range 1-100 |
| Password | `--password` | Prompted (optional) | None | Accepted but not used by Minecraft templates. Use RCON password instead (see below). |
| Auto-backup | `--auto-backup` | Prompted (Y/n) | Enabled (interactive), disabled (CLI) | Boolean |
| Version | N/A | Prompted | `latest` | Optional string, e.g. `1.20.4` |
| MOTD | N/A | Prompted | None | Optional string |
| Operators | N/A | Prompted | None | Comma-separated Minecraft usernames |

**Interactive-only parameters:** Version, MOTD, and Operators are only available through the interactive TUI. They are not exposed as CLI flags.

## Deploy Examples

### Interactive Mode

Launch ZPlay without arguments to enter the interactive TUI:

```bash
zplay
```

Select **Deploy** from the main menu, then **Minecraft**. The TUI walks through each parameter with prompts, menus, and sensible defaults. Port auto-increments if the default is already in use.

### CLI Mode

Deploy a vanilla server with defaults:

```bash
zplay deploy --game minecraft --variant vanilla --name survival \
  --memory 4Gi --node oracle1 --port 25565 --auto-backup
```

Deploy a Paper server with custom settings:

```bash
zplay deploy --game minecraft --variant paper --name smp \
  --memory 8Gi --node oracle2 --port 25565 --max-players 20 --auto-backup
```

Deploy a Forge server:

```bash
zplay deploy --game minecraft --variant forge --name modded \
  --memory 8Gi --node oracle1 --port 25566
```

## RCON (Remote Console)

RCON is always enabled on all Minecraft deployments.

- **RCON port:** 25575 (cluster-internal only, not exposed through Traefik)
- **Default password:** `zplay` (used when no custom RCON password is set)
- **Custom password:** When set, stored as a Kubernetes Secret with key `rcon-password`. The Secret template only renders if a custom RCON password is configured.

### Connecting to RCON

RCON is cluster-internal only. To connect from outside the cluster, use `kubectl port-forward`:

```bash
# Forward RCON port to localhost
kubectl port-forward svc/<server-name> 25575:25575

# In another terminal, connect with an RCON client
# Using mcrcon as an example:
mcrcon -H 127.0.0.1 -P 25575 -p zplay
```

Replace `<server-name>` with the name of your Minecraft server and `zplay` with your custom RCON password if one was set.

## Data Path

All server data is stored at `/data` inside the container. This path is backed by a Kubernetes PersistentVolumeClaim (PVC), so world data, configuration files, and server state persist across restarts, stops, and pod rescheduling.

## Networking

| Port | Protocol | Purpose | External access |
|------|----------|---------|-----------------|
| 25565 | TCP | Game traffic | Exposed via Traefik TCP IngressRouteTCP |
| 25575 | TCP | RCON | Cluster-internal only (Service port, no Traefik route) |

**Connection address:** `play.zyrak.cloud:25565`

Only the game port is routed through Traefik. RCON remains internal to the cluster and requires `kubectl port-forward` for external access.

## Health Probes

Health checks use the `mc-health` command built into the itzg/minecraft-server image.

| Probe | Method | Initial delay | Period | Failure threshold |
|-------|--------|---------------|--------|-------------------|
| Liveness | `exec: mc-health` | 120s | 30s | 5 |
| Readiness | `exec: mc-health` | 60s | 10s | 5 |

The liveness probe has a longer initial delay (120s) to account for Minecraft's startup time, which includes world generation on first boot.

## Resource Defaults

| Resource | Request | Limit |
|----------|---------|-------|
| Memory | 4Gi | 8Gi |
| CPU | 500m | 2 |

These are the same defaults used for Terraria servers. Memory can be adjusted with the `--memory` flag, but note that the `raspberry` node enforces a maximum of `4Gi`.

## Known Limitations

- **No custom map upload via CLI.** World files cannot be uploaded through ZPlay. Use `kubectl cp` to transfer world data into the `/data` volume manually.
- **No seed configuration.** World seed cannot be set through ZPlay parameters.
- **No whitelist management.** Player whitelisting must be configured through RCON or by editing `whitelist.json` directly.
- **No difficulty or gamemode configuration.** These must be set through RCON commands (`/difficulty`, `/gamemode`) or server.properties.
- **Version, MOTD, and Operators are interactive-only.** These parameters are not available as CLI flags and can only be set through the TUI.
- **Maximum 2 concurrent Minecraft servers.** Limited by the available port range (25565, 25566).

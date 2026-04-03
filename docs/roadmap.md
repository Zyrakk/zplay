# Roadmap

This document covers the development history of ZPlay and outlines planned work.

## Completed Phases

### Phase 0 - Foundation

Established the core project structure and first game support.

- Go module layout with `cmd/` and `internal/` packages
- CLI skeleton built on Bubble Tea for interactive TUI
- Configuration management via `~/.zplay/config.yaml`
- Kubernetes client wrapper around `kubectl`
- Game interface and registry pattern for extensibility
- Basic Terraria server implementation

### Phase 1 - Robustness

Hardened input handling, error reporting, and state management.

- Input validation for server names, memory limits, and ports
- Improved error handling across CLI and Kubernetes operations
- Server state persistence via `~/.zplay/servers.yaml`
- Node existence validation before deployment

### Phase 2 - Terraria Variants

Extended Terraria support with modded server variants.

- tModLoader variant using the `jacobsmile/tmodloader1.4` image
- Variant-specific container images, environment variables, and probe configurations
- x86 node affinity constraint for tModLoader (lake)
- Memory warnings for modded server configurations

### Phase 3 - Backups and Restore

Added data protection through backup and restore workflows.

- Manual backup command producing `tar.gz` archives to `/mnt/das/zplay-backups`
- Automated backups via Kubernetes CronJob with configurable schedule and retention
- Restore from backup: stop server, clear data, extract archive, restart
- World save flush before backup to ensure data consistency

### Phase 4 - Usability

Improved day-to-day operations and user experience.

- Cluster reconciliation on list (adopt orphaned servers, clean stale local state)
- Direct CLI subcommands with flags: `deploy`, `list`, `delete`, `start`, `stop`, `backup`, `status`
- Detailed per-server status view with graceful N/A fallbacks
- Terminal reset after Bubble Tea exits (`stty sane`)
- Console attach via `tmux` with clean detach
- Log follow mode with clean Ctrl+C exit
- Exec probes replacing TCP probes to eliminate connection spam
- Cleanup command for orphaned namespaces

### Phase 5 - Minecraft

Added Minecraft as the second supported game with three variants.

- Vanilla, Paper, and Forge variants using the `itzg/minecraft-server` image
- RCON support (always enabled, configurable password)
- Version, MOTD, and operators configuration in interactive mode
- Full template set (9 Kubernetes resource templates)
- Comprehensive test coverage for template rendering and validation
- Web dashboard as a separate binary using `client-go`

## Current State

ZPlay is at **v0.5.0** with the following capabilities:

- **Games:** 2 supported games (Terraria, Minecraft) across 5 total variants
- **Lifecycle:** Full server management -- deploy, start, stop, delete, backup, restore, status, console, logs, cleanup
- **Interfaces:** Interactive TUI mode and direct CLI subcommands
- **Dashboard:** Web-based dashboard component for server monitoring

## Phase 6 - Operations (Planned)

- `zplay list --sync` for non-interactive cluster reconciliation
- `zplay restore <name> --backup <file>` for non-interactive restore
- `status --json` for structured output suitable for scripting and automation
- End-to-end automated tests running against a staging cluster

## Future Ideas

### New Games

- Vintage Story
- Factorio
- Valheim

### Monitoring

- Metrics export to VictoriaMetrics
- Grafana dashboards for game server health and resource usage
- Alerting on server downtime

### Game-Specific Enhancements

- Minecraft: expose difficulty, gamemode, whitelist, and seed via CLI flags
- Terraria: TShock variant support
- Terraria: seed configuration

### General Improvements

- Custom world and map upload via CLI

# ZPlay

A CLI tool for deploying and managing game servers on Kubernetes.

Version: `0.5.0`

## Overview

ZPlay is a Go CLI that deploys and manages game servers on a Kubernetes (k3s) cluster. It provides both an interactive terminal menu (powered by Bubble Tea) and direct CLI subcommands with flags. ZPlay handles the full server lifecycle: deploy, start/stop, backup/restore, status, console, logs, and cleanup.

## Supported Games

| Game | Variants | Status | Ports |
|------|----------|--------|-------|
| Terraria | vanilla, tModLoader | Stable | 7777, 7778 |
| Minecraft | vanilla, paper, forge | Stable | 25565, 25566 |

## Requirements

- Go 1.22+
- `kubectl` configured against your cluster
- Kubernetes cluster with Traefik TCP entrypoints for game ports
- `tmux` (optional, for clean console detach)

## Installation

### Option A: Build and run locally

```bash
git clone https://github.com/Zyrakk/zplay.git
cd zplay
make deps
make dev
```

### Option B: System-wide install

```bash
git clone https://github.com/Zyrakk/zplay.git
cd zplay
make install
```

Requires sudo. Builds and copies the binary to `/usr/local/bin`.

### Option C: User-local install

```bash
make deps
make build
mkdir -p "$HOME/.local/bin"
cp dist/zplay "$HOME/.local/bin/zplay"
```

Make sure `~/.local/bin` is in your `PATH`.

## Quick Start

Launch the interactive menu (no arguments):

```bash
zplay
```

Or deploy directly with CLI flags:

```bash
# Deploy a Terraria server
zplay deploy --game terraria --variant vanilla --name survival \
  --memory 4Gi --node oracle1 --port 7777 --auto-backup

# Deploy a Minecraft server
zplay deploy --game minecraft --variant paper --name mc1 \
  --memory 4Gi --node oracle1 --port 25565 --auto-backup
```

Common operations:

```bash
zplay list                    # List all servers
zplay status survival         # Detailed server status
zplay backup survival         # Manual backup
zplay stop survival           # Stop server (preserves data)
zplay start survival          # Start server
```

## Documentation

- [Terraria Server Guide](docs/terraria.md) -- Variants, parameters, mod installation
- [Minecraft Server Guide](docs/minecraft.md) -- Variants, parameters, RCON
- [CLI Reference](docs/cli-reference.md) -- All commands, flags, and examples
- [Configuration](docs/configuration.md) -- config.yaml schema and defaults
- [Operations Guide](docs/operations.md) -- Backup, restore, reconciliation, cleanup
- [Infrastructure](docs/infrastructure.md) -- Cluster setup, Traefik, storage
- [Architecture](docs/architecture.md) -- Code structure, extending ZPlay
- [Roadmap](docs/roadmap.md) -- Project history and future plans

## Development

```bash
make dev        # Run in development mode
make test       # Run tests
make build      # Build for current platform
make build-all  # Build for all platforms
make lint       # Run linter
make fmt        # Format code
```

## License

MIT

# ZPlay

Game server manager for Kubernetes. Deploy, manage, and monitor game servers on your k3s cluster.

## Features

- 🎮 **Interactive CLI** - Easy-to-use menu-driven interface
- 🚀 **Quick Deploy** - Spin up game servers in seconds
- 📊 **Server Management** - List, delete, view logs
- 🖥️ **Console Access** - Attach to server console directly
- 🔌 **ZCloud Integration** - Works with your existing zcloud setup

## Supported Games

- ✅ Terraria
- 🔜 Minecraft (Paper, Fabric, Forge)

## Requirements

- Go 1.22+
- kubectl
- Access to a Kubernetes cluster (via zcloud or direct kubeconfig)
- Traefik ingress controller (included with k3s)

## Installation

```bash
# Clone the repository
git clone https://github.com/Zyrakk/zplay.git
cd zplay

# Install dependencies
make deps

# Build and install
make install
```

## Usage

Make sure you're connected to your cluster first:

```bash
# If using zcloud
zcloud login
```

Then run zplay:

```bash
zplay
```

You'll see an interactive menu:

```
╔═══════════════════════════════════════╗
║     ZPlay - Game Server Manager       ║
╚═══════════════════════════════════════╝

▸ Deploy server
  List servers
  Delete server
  Server console
  View logs
  Exit
```

### Deploy a Server

1. Select "Deploy server"
2. Choose the game (Terraria, etc.)
3. Enter server name, memory, world size, etc.
4. Confirm and wait for deployment

```
Server ready!
Connect to: play.zyrak.cloud:7777
```

### List Servers

Shows all deployed servers with their status:

```
NAME            GAME         PORT     MEMORY     STATUS     ADDRESS
───────────────────────────────────────────────────────────────────
survival        terraria     7777     4Gi        Running    play.zyrak.cloud:7777
creative        terraria     7778     4Gi        Running    play.zyrak.cloud:7778
```

### Server Console

Attach to a server's interactive console:

```bash
# Through the menu
zplay → Server console → Select server

# Detach with Ctrl+P, Ctrl+Q
```

### View Logs

Stream or view server logs:

```bash
zplay → View logs → Select server → Follow? [Y/n]
```

## Configuration

Configuration is stored in `~/.zplay/config.yaml`:

```yaml
domain: play.zyrak.cloud
kubeconfig: ~/.zcloud/kubeconfig
node_selector: ""  # Optional: pin servers to a specific node
data_path: ~/.zplay
```

## Traefik Configuration

ZPlay creates IngressRouteTCP resources for each server. You need to configure Traefik entrypoints for the game ports.

Add to your Traefik Helm values or HelmChartConfig:

```yaml
ports:
  zplay-7777:
    port: 7777
    expose: true
    exposedPort: 7777
    protocol: TCP
  zplay-7778:
    port: 7778
    expose: true
    exposedPort: 7778
    protocol: TCP
  # Add more ports as needed
```

Or use a port range approach (see docs).

## File Structure

```
~/.zplay/
├── config.yaml      # Configuration
└── servers.yaml     # Deployed servers state
```

## Development

```bash
# Run in dev mode
make dev

# Run tests
make test

# Build for all platforms
make build-all
```

## License

MIT

## Author

Zyrak - [zyrak.cloud](https://zyrak.cloud)

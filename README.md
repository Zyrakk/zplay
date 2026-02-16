# ZPlay

ZPlay is an interactive CLI to deploy and operate game servers on Kubernetes (k3s).

## Project Status

Current status (February 16, 2026):

- Phase 0 - Foundation: completed.
  - Project compiles and deploys Terraria vanilla successfully.
- Phase 1 - Robustness: completed.
  - Node selection in deploy flow.
  - Deploy validations (name, port, memory, node limits).
  - Passwords moved to Kubernetes Secrets.
  - Start/Stop servers without deleting data.
  - Terraria difficulty selection (Classic/Expert/Master/Journey).
- Phase 2+ (mods, backups, Minecraft expansion): planned in `docs/roadmap.md`.

## Supported Games

- Terraria (vanilla): stable.
- Minecraft: not implemented yet.

## Requirements

- Go 1.22+
- `kubectl` configured against your cluster
- Access to your Kubernetes cluster (`zcloud login` or direct kubeconfig)
- Traefik with TCP entrypoints for game ports

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

By default, ZPlay reads kubeconfig from `~/.zcloud/kubeconfig`.

## Main Menu

```text
Deploy server
List servers
Delete server
Start/Stop server
Server console
View logs
Exit
```

## Deploy Flow (Terraria)

Deploy prompts include:

1. Server name
2. Node selection
   - `oracle1`
   - `oracle2`
   - `raspberry`
   - `Auto`
3. Server type (vanilla / tModLoader selector present)
4. Memory
5. World size
6. Difficulty
   - Classic (`0`)
   - Expert (`1`)
   - Master (`2`)
   - Journey (`3`)
7. Max players
8. Optional password
9. Port

Pre-deploy summary shows selected node, world size, difficulty, and other settings.

## Phase 1 Validations

Before deploy confirmation, ZPlay validates:

- Name must be RFC1123-compatible (`2-20` chars, lowercase, digits, `-`)
- Port must be allowed for the game entrypoints
  - Terraria: `7777`, `7778`
- Port must not already be used by another server
- Memory format must match `^\d+[GM]i$` (examples: `4Gi`, `512Mi`)
- `raspberry` node max memory is `4Gi`

## Password Handling (Kubernetes Secrets)

When a password is provided:

- ZPlay creates a Secret: `<server>-secret`
- Deployment references password via `valueFrom.secretKeyRef`
- Password is not stored as plaintext env value in Deployment manifests

When password is empty:

- No Secret is rendered/applied.

## Start/Stop Without Deletion

`Start/Stop server` scales deployment replicas:

- Running server -> Stop -> replicas `0`
- Stopped server -> Start -> replicas `1` and wait for readiness

PVC/world data is preserved because namespace/PVC are not deleted.

## List Output

`List servers` includes:

- `NODE` column (`oracle1`, `oracle2`, `raspberry`, or `auto`)
- Real status, including `Stopped` when deployment replicas are `0`

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

# CLI Reference

ZPlay v0.5.0 operates in two modes: an interactive TUI launched with no arguments, and direct CLI subcommands for scripting and automation.

---

## Interactive Mode

```
zplay
```

Running `zplay` with no arguments starts the interactive terminal UI.

**Menu options:**

- Deploy server
- List servers
- Start server
- Stop server
- Server status
- Backup server
- Restore backup
- Delete server
- Server console
- View logs
- Cleanup resources
- Exit

**Navigation:**

| Key | Action |
|---|---|
| Up / k | Move selection up |
| Down / j | Move selection down |
| Enter | Select option |
| q | Quit |

---

## Commands

### version

Print the current ZPlay version.

```
zplay version
```

**Example:**

```
$ zplay version
zplay v0.5.0
```

---

### deploy

Deploy a new game server to the cluster.

```
zplay deploy --game <game> --variant <variant> --name <name> --memory <memory> \
             --node <node|auto> --port <port> [--password <password>] \
             [--max-players <n>] [--world-size <size>] [--difficulty <level>] \
             [--auto-backup]
```

**Flags:**

| Flag | Required | Default | Description |
|---|---|---|---|
| `--game` | Yes | -- | Game type: `terraria`, `minecraft` |
| `--variant` | No | `vanilla` | Server variant: `vanilla`, `tmodloader`, `paper`, `forge` |
| `--name` | Yes | -- | Server name (RFC 1123 subdomain, 2-20 lowercase alphanumeric characters and hyphens) |
| `--memory` | Yes | -- | Memory request, e.g. `2Gi`, `512Mi` |
| `--node` | Yes | -- | Target node hostname, or `auto` to let the Kubernetes scheduler decide |
| `--port` | Yes | -- | Service port (must be an allowed entrypoint for the game) |
| `--password` | No | -- | Server password |
| `--max-players` | No | `8` | Maximum number of players |
| `--world-size` | No | `medium` | World size (Terraria only): `small`, `medium`, `large` |
| `--difficulty` | No | `0` | Difficulty level (Terraria only): `0`, `1`, `2`, `3` |
| `--auto-backup` | No | `false` | Enable daily automatic backups |

**Terraria example:**

```
$ zplay deploy \
    --game terraria \
    --variant vanilla \
    --name terraria-chill \
    --memory 2Gi \
    --node auto \
    --port 7777 \
    --max-players 10 \
    --world-size large \
    --difficulty 1 \
    --auto-backup
```

**Minecraft example:**

```
$ zplay deploy \
    --game minecraft \
    --variant paper \
    --name mc-survival \
    --memory 4Gi \
    --node lake \
    --port 25565 \
    --max-players 16 \
    --auto-backup
```

**Notes:**

- The `--world-size` and `--difficulty` flags apply to Terraria only and are ignored for Minecraft deployments.
- Some Minecraft configuration parameters (e.g. gamemode, seed) are available only through the interactive TUI and cannot be set via CLI flags.
- The memory limit is automatically set to 2x the requested memory value.
- See the Validation Rules section below for port, name, and node constraints.

---

### list

List all deployed game servers.

```
zplay list [--json]
```

**Flags:**

| Flag | Required | Default | Description |
|---|---|---|---|
| `--json` | No | `false` | Output in JSON format |

**Example:**

```
$ zplay list
```

```
$ zplay list --json
```

---

### delete

Delete a deployed server. Requires explicit confirmation.

```
zplay delete <name> --yes
```

**Flags:**

| Flag | Required | Default | Description |
|---|---|---|---|
| `--yes` | Yes | -- | Confirm deletion (safety flag, must be provided) |

**Example:**

```
$ zplay delete terraria-chill --yes
```

---

### stop

Stop a running server without deleting it.

```
zplay stop <name>
```

**Example:**

```
$ zplay stop mc-survival
```

---

### start

Start a previously stopped server.

```
zplay start <name>
```

**Example:**

```
$ zplay start mc-survival
```

---

### backup

Create a backup of a server.

```
zplay backup <name>
```

**Example:**

```
$ zplay backup terraria-chill
```

---

### status

Show the current status of a server.

```
zplay status <name>
```

**Example:**

```
$ zplay status mc-survival
```

---

### cleanup

Remove orphaned or unused cluster resources.

```
zplay cleanup [--yes] [--dry-run] [--json]
```

**Flags:**

| Flag | Required | Default | Description |
|---|---|---|---|
| `--yes` | No | `false` | Skip confirmation prompt |
| `--dry-run` | No | `false` | Show what would be removed without making changes |
| `--json` | No | `false` | Output in JSON format |

**Example:**

```
$ zplay cleanup --dry-run
```

```
$ zplay cleanup --yes
```

---

## Validation Rules

The following validation rules are enforced during `zplay deploy`:

- **Server name format** -- Must match `^[a-z][a-z0-9-]{0,18}[a-z0-9]$`. This means 2-20 characters, starting with a lowercase letter, ending with a lowercase letter or digit, and containing only lowercase letters, digits, and hyphens. Conforms to RFC 1123 subdomain naming.

- **Memory format** -- Must match `^\d+[GM]i$`. Valid examples: `512Mi`, `2Gi`, `4Gi`.

- **Memory limit inference** -- The memory limit is automatically set to 2x the requested memory value. For example, requesting `2Gi` results in a limit of `4Gi`.

- **Port constraints per game** -- Each game has a fixed set of allowed ports:
  - Terraria: `7777`, `7778`
  - Minecraft: `25565`, `25566`

- **Port uniqueness** -- No two servers can share the same port. Deployment will fail if the requested port is already in use.

- **Node validation** -- The target node must exist in the cluster. ZPlay validates node existence before proceeding with deployment. Use `auto` to let ZPlay select a node automatically.

- **Node-specific limits:**
  - Nodes with the `raspberry` hostname are capped at `4Gi` memory.
  - The `tmodloader` variant is forced to deploy on the `lake` node regardless of the `--node` flag.

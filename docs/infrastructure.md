# Infrastructure Requirements

ZPlay deploys game servers on a k3s Kubernetes cluster. This document describes the cluster infrastructure, networking, and storage requirements for running ZPlay.

## Cluster Requirements

**Required:**

- k3s (or any conformant Kubernetes distribution)
- `kubectl` configured and accessible from the machine running ZPlay

**Optional:**

- `metrics-server` -- enables CPU and memory reporting in the `zplay status` command
- `tmux` -- allows clean console detach when running interactive sessions

## Node Inventory

| Node | RAM | Architecture | Role | Notes |
|------|-----|--------------|------|-------|
| oracle1 | 24 GB | ARM64 | Worker | Recommended scheduling target, 5TB DAS at `/mnt/das` |
| oracle2 | 24 GB | ARM64 | Worker | Secondary worker, 5TB DAS at `/mnt/das` |
| raspberry | 8 GB | ARM64 | Worker | Light servers only, hard ceiling of 4Gi per server |
| lake | 16 GB | AMD64 | Control plane + x86 workloads | Required for tModLoader (x86-only binary) |

**Scheduling constraints:**

- **tModLoader** must run on `lake` because it ships only an x86-64 binary. ARM64 nodes cannot run it.
- **raspberry** has 8 GB total RAM shared with the OS and other workloads. Individual game servers scheduled here must not request more than 4Gi.
- For all other game servers, prefer `oracle1` or `oracle2` for their higher memory headroom and attached storage.

## Storage

### Storage Classes

| StorageClass | Backing | Capacity | Use |
|--------------|---------|----------|-----|
| `nfs-shared` | LVM | 5 TB | Game server PVCs (default) |
| `nfs-nvme` | NVMe | 500 GB | Reserved for monitoring -- do not use for game servers |
| `local-path` | Variable | Variable | Not suitable for game servers (no cross-node access) |

### PVC Defaults

- Default size: **10Gi**
- Default class: configurable in `config.yaml`

Both values can be overridden per server via the `storage.size` and `storage.class` fields in the server configuration.

## Traefik TCP Entrypoints

ZPlay exposes game servers through Traefik **IngressRouteTCP** resources (not standard HTTP Ingress). Each game server maps to a fixed TCP entrypoint defined in Traefik's configuration. The mapping between ports and entrypoint names is hardcoded in `PortToEntrypoint` (see `game.go`).

### Port Mapping

| Port | Entrypoint Name | Game |
|------|----------------|------|
| 7777 | `terraria1` | Terraria server 1 |
| 7778 | `terraria2` | Terraria server 2 |
| 25565 | `minecraft1` | Minecraft server 1 |
| 25566 | `minecraft2` | Minecraft server 2 |

### HelmChartConfig

Apply the following configuration to register these entrypoints with the k3s-managed Traefik instance:

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: traefik
  namespace: kube-system
spec:
  valuesContent: |-
    ports:
      terraria1:
        port: 7777
        expose:
          default: true
        exposedPort: 7777
        protocol: TCP
      terraria2:
        port: 7778
        expose:
          default: true
        exposedPort: 7778
        protocol: TCP
      minecraft1:
        port: 25565
        expose:
          default: true
        exposedPort: 25565
        protocol: TCP
      minecraft2:
        port: 25566
        expose:
          default: true
        exposedPort: 25566
        protocol: TCP
```

Place this file at:

```
/var/lib/rancher/k3s/server/manifests/traefik-config.yaml
```

k3s automatically applies manifests in this directory. After placing the file, verify the ports are exposed on the Traefik LoadBalancer service:

```bash
kubectl get svc -n kube-system traefik -o yaml | grep -A20 ports
```

You should see entries for ports 7777, 7778, 25565, and 25566 in the output.

## DNS Configuration

The domain `play.zyrak.cloud` must resolve to the cluster's ingress IP address (the external IP of the Traefik LoadBalancer service).

To find the current ingress IP:

```bash
kubectl get svc -n kube-system traefik -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Create or update the DNS A record so that `play.zyrak.cloud` points to this IP.

## Backup Storage

Game server backups are stored on the host filesystem, separate from game data storage for failure isolation.

- **Host path:** `/mnt/das/zplay-backups`
- **Backup node:** `oracle1` (default)
- **Separation:** Backup storage must not share a volume or StorageClass with live game data. This ensures a storage failure affecting game PVCs does not also destroy backups.

Ensure the backup directory exists on the target node:

```bash
mkdir -p /mnt/das/zplay-backups
```

## Initial Setup Checklist

Complete these steps once when bringing up a new cluster or reinstalling ZPlay:

1. **Configure Traefik entrypoints** -- Place the HelmChartConfig YAML at `/var/lib/rancher/k3s/server/manifests/traefik-config.yaml` on the control plane node. Verify ports appear on the Traefik service.
2. **Verify DNS** -- Confirm `play.zyrak.cloud` resolves to the Traefik LoadBalancer external IP.
3. **Create backup directory** -- Run `mkdir -p /mnt/das/zplay-backups` on the backup node (default `oracle1`).
4. **Install ZPlay** -- Follow the installation instructions in the project README.

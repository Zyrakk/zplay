# ZPlay - TODO & Roadmap

## Phase 1: Core Terraria (Current)

### ✅ Completed
- [x] Project structure
- [x] Basic CLI with bubbletea
- [x] Config management
- [x] K8s client wrapper
- [x] Games interface (extensible)
- [x] Terraria implementation
- [x] Deploy command
- [x] List command
- [x] Delete command
- [x] Console command
- [x] Logs command
- [x] Server state persistence

### 🔧 To Fix/Improve

#### High Priority
- [ ] **Test compilation** - Run `go mod tidy` and fix any import issues
- [ ] **Traefik entrypoints** - Document/automate entrypoint creation
  - Currently requires manual Traefik config per port
  - Options: 
    - Use NodePort instead of IngressRouteTCP
    - Create a port range in Traefik config
    - Use a single high port range (7770-7780)
- [ ] **Password handling** - Use Kubernetes Secrets instead of env vars
  ```yaml
  # Create secret template
  apiVersion: v1
  kind: Secret
  metadata:
    name: {{.Name}}-secret
  stringData:
    password: {{.Password}}
  ```
- [ ] **Error handling** - Better error messages for common failures
  - Cluster not reachable
  - Namespace already exists
  - Port already in use

#### Medium Priority
- [ ] **Validation improvements**
  - Check if port is already in use before deploy
  - Validate memory format (must end in Gi/Mi)
  - Check node selector exists
- [ ] **Server start/stop** - Scale deployment to 0/1 without deleting
  ```go
  func (c *Client) ScaleDeployment(namespace, deployment string, replicas int) error
  ```
- [ ] **Status command** - More detailed server info
  - Pod events
  - Resource usage
  - Uptime
- [ ] **Backup/Restore** - Save world data
  - `kubectl cp` to local
  - Store in ~/.zplay/backups/

#### Low Priority
- [ ] **Non-interactive mode** - CLI flags for scripting
  ```bash
  zplay deploy --game terraria --name survival --memory 4Gi --world-size large
  zplay delete survival --yes
  zplay list --json
  ```
- [ ] **Server restart** - Quick restart without full redeploy
- [ ] **Config edit** - Change server settings after deploy (memory, players)

---

## Phase 2: Terraria Polish

### Features
- [ ] **Difficulty selection** - Easy/Normal/Expert/Master/Journey
- [ ] **World seed** - Custom seed support
- [ ] **Auto-backup** - CronJob for periodic backups
- [ ] **TShock support** - Alternative image with TShock server
- [ ] **Mods support** - tModLoader image option

### Monitoring
- [ ] **Basic metrics** - CPU/Memory from kubectl top
- [ ] **Health status** - More detailed pod health info
- [ ] **Player count** - Parse logs for connected players (if possible)

### UX
- [ ] **Colors/themes** - Customizable CLI colors
- [ ] **Progress bars** - Show deployment progress
- [ ] **Notifications** - Optional desktop notifications when server ready

---

## Phase 3: Minecraft

### Implementation
- [ ] **Minecraft game module** - `internal/games/minecraft/`
- [ ] **Helm integration** - Use itzg/minecraft-server-charts
  - OR create own templates (simpler, more control)
- [ ] **Server types**
  - Paper (default, optimized)
  - Fabric (mods)
  - Forge (mods)
  - Vanilla
- [ ] **Version selection** - List available versions
- [ ] **RCON support** - Remote console via RCON
- [ ] **Ops management** - Add/remove operators

### Minecraft-Specific Features
- [ ] **Whitelist management** - Add/remove players
- [ ] **World download** - Download world to local
- [ ] **Plugin/Mod management** - For Paper/Fabric/Forge
- [ ] **Server properties** - Edit server.properties via CLI

---

## Phase 4: Advanced Features

### Multi-Game Support
- [ ] **Factorio**
- [ ] **Valheim**
- [ ] **Vintage Story** (ya tienes en k3s-gameservers)
- [ ] **Satisfactory**
- [ ] **7 Days to Die**

### Infrastructure
- [ ] **Resource quotas** - Limit total resources per game
- [ ] **Scheduled servers** - Auto start/stop at certain times
- [ ] **Multi-cluster** - Support multiple k8s clusters
- [ ] **Web UI** - Optional web dashboard (future, maybe)

### Observability
- [ ] **VictoriaMetrics integration** - Push metrics
- [ ] **Grafana dashboards** - Pre-built dashboards
- [ ] **Alerting** - Notifications when server down

---

## Technical Debt

### Code Quality
- [ ] **Unit tests** - Test game validation, config, etc.
- [ ] **Integration tests** - Test against kind/k3d cluster
- [ ] **CI/CD** - GitHub Actions for build/test/release
- [ ] **Documentation** - GoDoc comments

### Refactoring
- [ ] **Better template system** - Consider using Helm charts internally
- [ ] **Plugin system** - Allow external game definitions
- [ ] **Config validation** - JSON Schema for config files

---

## Traefik Port Configuration

### Option 1: Static Ports (Simple)
Add to k3s Traefik HelmChartConfig:

```yaml
# /var/lib/rancher/k3s/server/manifests/traefik-config.yaml
apiVersion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: traefik
  namespace: kube-system
spec:
  valuesContent: |-
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
      zplay-25565:
        port: 25565
        expose: true
        exposedPort: 25565
        protocol: TCP
```

### Option 2: NodePort (No Traefik config needed)
Change service template to use NodePort:

```yaml
apiVersion: v1
kind: Service
spec:
  type: NodePort
  ports:
    - port: 7777
      nodePort: {{.Port}}
```

**Pros**: No Traefik config needed
**Cons**: Port range limited to 30000-32767 by default

### Option 3: LoadBalancer with MetalLB
If you have MetalLB installed, use LoadBalancer type.

---

## Notes

### Image Options for Terraria
- `passivelemon/terraria-docker:terraria-latest` - Current, good
- `ryshe/terraria:latest` - Alternative, also popular
- `jacobsmile/tmodloader:latest` - For modded Terraria

### Useful Commands for Testing
```bash
# Check if zcloud is connected
zcloud status

# Manual namespace cleanup
kubectl delete namespace zplay-testserver

# Check Traefik entrypoints
kubectl get svc -n kube-system traefik

# Port forward for testing without Traefik
kubectl port-forward -n zplay-survival deployment/survival-terraria 7777:7777
```

---

## Questions to Resolve

1. **Storage class**: Using `local-path` - is this available on all your nodes?
2. **Node selector**: Should we auto-detect the best node based on resources?
3. **Port management**: How many concurrent servers do you realistically need?
4. **Backup storage**: Local or cloud (S3/Cloudflare R2)?

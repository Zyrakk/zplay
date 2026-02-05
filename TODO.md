# ZPlay - TODO & Roadmap

## Infraestructura ZCloud

### Nodos disponibles
| Nodo | RAM | Uso recomendado | Notas |
|------|-----|-----------------|-------|
| **oracle1** | 24GB | Minecraft, Terraria con mods | ARM64, 5TB en /mnt/das |
| **oracle2** | 24GB | Minecraft, Terraria con mods | ARM64, 5TB en /mnt/das |
| **raspberry** | 8GB | Terraria vanilla (4GB max) | ARM64, 4TB en /mnt/local |
| **n150** | 16GB | ❌ No usar | Control plane, Wazuh, VictoriaMetrics |

### Storage Classes
| StorageClass | Capacidad | Uso |
|--------------|-----------|-----|
| `nfs-shared` | 5TB LVM | **Servers (worlds)** - HA, accesible desde cualquier nodo |
| `nfs-nvme` | 500GB | ❌ Reservado para VictoriaMetrics, Wazuh (alto I/O) |
| `local-path` | Variable | ❌ No usar para game servers |

### Backups
- **Destino**: `/mnt/das` en oracle1 u oracle2
- **Razón**: Separado del storage principal, si nfs-shared falla las backups están a salvo

### Puertos asignados
| Puerto | Juego | Server |
|--------|-------|--------|
| 7777 | Terraria | Server 1 (vanilla) |
| 7778 | Terraria | Server 2 (mods) |
| 25565 | Minecraft | Server 1 |
| 25566 | Minecraft | Server 2 |

**Dominio**: `play.zyrak.cloud`

---

## Phase 1: Core Terraria

### ✅ Completado
- [x] Estructura del proyecto
- [x] CLI básico con bubbletea
- [x] Gestión de configuración
- [x] Cliente K8s wrapper
- [x] Interface de juegos (extensible)
- [x] Implementación Terraria
- [x] Comando deploy
- [x] Comando list
- [x] Comando delete
- [x] Comando console
- [x] Comando logs
- [x] Persistencia de estado de servers

### 🔧 Por hacer (Prioridad Alta)

#### Compilación y testing
- [ ] Ejecutar `go mod tidy` y arreglar imports
- [ ] Compilar y probar localmente
- [ ] Test básico de deploy en ZCloud

#### Actualizar templates con tu infra
- [ ] Cambiar storageClass a `nfs-shared` en `volume.yaml`
  ```yaml
  storageClassName: nfs-shared
  ```
- [ ] Añadir nodeSelector por defecto a `deployment.yaml`
  ```yaml
  nodeSelector:
    kubernetes.io/hostname: oracle1  # o oracle2
  ```
- [ ] Hacer nodeSelector configurable en deploy

#### Configurar Traefik
Añadir al HelmChartConfig de Traefik (`/var/lib/rancher/k3s/server/manifests/traefik-config.yaml`):

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: traefik
  namespace: kube-system
spec:
  valuesContent: |-
    ports:
      # Terraria
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
      # Minecraft (futuro)
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

Después aplicar:
```bash
# k3s recarga automáticamente, pero si no:
sudo systemctl restart k3s
```

#### Actualizar template de ingress
Cambiar `ingress.yaml` para usar los entrypoints fijos:
```yaml
spec:
  entryPoints:
    - terraria1  # o terraria2 según el puerto
```

O mejor, mapear puerto a entrypoint en el código:
```go
func portToEntrypoint(game string, port int) string {
    switch game {
    case "terraria":
        if port == 7777 { return "terraria1" }
        if port == 7778 { return "terraria2" }
    case "minecraft":
        if port == 25565 { return "minecraft1" }
        if port == 25566 { return "minecraft2" }
    }
    return fmt.Sprintf("zplay-%d", port)
}
```

### 🔧 Por hacer (Prioridad Media)

#### Selección de nodo en deploy
Añadir al menú de deploy:
```
Select node:
  1) oracle1 (24GB RAM) - recommended
  2) oracle2 (24GB RAM)
  3) raspberry (8GB RAM) - light servers only
```

Validar que Terraria con mods no se despliegue en raspberry (4GB max).

#### Secrets para passwords
Crear template `secret.yaml`:
```yaml
{{- if .Password}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Name}}-secret
  namespace: zplay-{{.Name}}
type: Opaque
stringData:
  password: "{{.Password}}"
{{- end}}
```

Actualizar `deployment.yaml`:
```yaml
{{- if .Password}}
- name: PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{.Name}}-secret
      key: password
{{- end}}
```

#### Validaciones
- [ ] Verificar que el puerto no está en uso antes de deploy
- [ ] Validar formato de memoria (debe terminar en Gi/Mi)
- [ ] Si raspberry seleccionada, limitar memoria a 4Gi máximo
- [ ] Verificar que el nodo existe y tiene suficiente RAM

#### Start/Stop servers
Escalar deployment a 0/1 sin eliminar:
```go
func (c *Client) ScaleDeployment(namespace, deployment string, replicas int) error {
    cmd := c.kubectl("scale", "deployment", deployment,
        "-n", namespace,
        fmt.Sprintf("--replicas=%d", replicas))
    return cmd.Run()
}
```

Añadir al menú:
```
▸ Deploy server
  List servers
  Start/Stop server   # NUEVO
  Delete server
  ...
```

### 🔧 Por hacer (Prioridad Baja)

#### Modo no interactivo
```bash
zplay deploy --game terraria --name vanilla --memory 4Gi --node oracle1 --port 7777
zplay delete vanilla --yes
zplay list --json
zplay stop vanilla
zplay start vanilla
```

#### Status detallado
```bash
zplay status vanilla
# Output:
# Server: vanilla
# Game: Terraria
# Status: Running
# Node: oracle1
# Memory: 4Gi / 4Gi used
# CPU: 0.5 / 2 cores
# Uptime: 3d 14h
# Players: 2 connected
# Port: play.zyrak.cloud:7777
```

---

## Phase 2: Terraria Polish

### Features
- [ ] **Dificultad** - Classic/Expert/Master/Journey
- [ ] **Seed del mundo** - Seed personalizada
- [ ] **TShock** - Imagen alternativa con TShock para admin avanzado
- [ ] **tModLoader** - Imagen con soporte de mods
  - Imagen: `jacobsmile/tmodloader:latest`

### Backups

#### Backup manual
```bash
zplay backup vanilla
# Copia /opt/terraria/config a /mnt/das/zplay-backups/vanilla-2024-01-15.tar.gz
```

Implementación:
```go
func (c *Client) Backup(namespace, pvcName, destPath string) error {
    // Crear job temporal que monta el PVC y hace tar a /mnt/das
    // O usar kubectl cp si el pod está running
}
```

#### Auto-backup con CronJob
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{.Name}}-backup
  namespace: zplay-{{.Name}}
spec:
  schedule: "0 4 * * *"  # 4 AM diario
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: alpine
              command: ["/bin/sh", "-c"]
              args:
                - |
                  tar -czf /backup/{{.Name}}-$(date +%Y%m%d).tar.gz /data
              volumeMounts:
                - name: data
                  mountPath: /data
                - name: backup
                  mountPath: /backup
          restartPolicy: OnFailure
          volumes:
            - name: data
              persistentVolumeClaim:
                claimName: {{.Name}}-pvc
            - name: backup
              hostPath:
                path: /mnt/das/zplay-backups
          nodeSelector:
            kubernetes.io/hostname: oracle1
```

---

## Phase 3: Minecraft

### Implementación base
- [ ] Módulo minecraft en `internal/games/minecraft/`
- [ ] Usar imagen `itzg/minecraft-server` (muy completa)
- [ ] Templates propios (más control que Helm)

### Server types
| Tipo | Imagen tag | Uso |
|------|-----------|-----|
| Paper | `TYPE=PAPER` | Default, optimizado |
| Fabric | `TYPE=FABRIC` | Mods client-side |
| Forge | `TYPE=FORGE` | Mods pesados |
| Vanilla | `TYPE=VANILLA` | Puro |

### Template deployment.yaml (Minecraft)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}-minecraft
  namespace: zplay-{{.Name}}
spec:
  template:
    spec:
      containers:
        - name: minecraft
          image: itzg/minecraft-server
          env:
            - name: EULA
              value: "TRUE"
            - name: TYPE
              value: "{{.ServerType}}"
            - name: VERSION
              value: "{{.Version}}"
            - name: MEMORY
              value: "{{.JavaMemory}}"
            - name: MAX_PLAYERS
              value: "{{.MaxPlayers}}"
            - name: MOTD
              value: "{{.MOTD}}"
            - name: OPS
              value: "{{.Ops}}"
            - name: ENABLE_RCON
              value: "true"
            - name: RCON_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.Name}}-secret
                  key: rcon-password
          ports:
            - containerPort: 25565
              name: minecraft
            - containerPort: 25575
              name: rcon
```

### Features específicas de Minecraft
- [ ] **RCON console** - Consola remota sin kubectl attach
- [ ] **Whitelist** - Gestionar lista blanca
- [ ] **Ops** - Añadir/quitar operadores
- [ ] **Plugins/Mods** - Descargar e instalar

---

## Phase 4: Extras

### Otros juegos (futuro lejano)
- [ ] Vintage Story (ya tienes en k3s-gameservers)
- [ ] Factorio
- [ ] Valheim

### Mejoras de infraestructura
- [ ] **Métricas en VictoriaMetrics** - Exportar uso de recursos
- [ ] **Dashboards Grafana** - Panel para game servers
- [ ] **Alertas** - Notificar si server caído

---

## Configuración inicial requerida

### 1. Traefik (hacer una vez)
```bash
# En el nodo N150 (control plane)
sudo nano /var/lib/rancher/k3s/server/manifests/traefik-config.yaml
# Pegar la configuración de arriba
# Guardar y esperar a que k3s recargue
```

Verificar:
```bash
kubectl get svc -n kube-system traefik -o yaml | grep -A20 ports
```

### 2. DNS (si no está)
Asegurar que `play.zyrak.cloud` apunta a la IP de tu cluster (o del nodo con Traefik).

### 3. Directorio de backups
```bash
# En oracle1
sudo mkdir -p /mnt/das/zplay-backups
sudo chmod 777 /mnt/das/zplay-backups

# En oracle2 (redundancia)
sudo mkdir -p /mnt/das/zplay-backups
sudo chmod 777 /mnt/das/zplay-backups
```

---

## Archivos a modificar

### volume.yaml
```yaml
# Cambiar storageClassName
spec:
  storageClassName: nfs-shared  # Era: local-path
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

### deployment.yaml
```yaml
# Añadir nodeSelector dinámico
spec:
  template:
    spec:
      {{- if .NodeSelector}}
      nodeSelector:
        kubernetes.io/hostname: {{.NodeSelector}}
      {{- end}}
```

### terraria.go
```go
// Añadir NodeSelector a ServerConfig desde el menú
// Validar memoria máxima según nodo seleccionado
```

### deploy.go
```go
// Añadir selección de nodo
fmt.Println("\nSelect node:")
fmt.Println("  1) oracle1 (24GB RAM) - recommended")
fmt.Println("  2) oracle2 (24GB RAM)")
fmt.Println("  3) raspberry (8GB RAM) - light servers only")
```

---

## Comandos útiles para testing

```bash
# Verificar conexión zcloud
zcloud status

# Ver namespaces de zplay
kubectl get ns | grep zplay

# Ver pods de un server
kubectl get pods -n zplay-vanilla

# Logs de un server
kubectl logs -f deployment/vanilla-terraria -n zplay-vanilla

# Port forward para test sin Traefik
kubectl port-forward -n zplay-vanilla deployment/vanilla-terraria 7777:7777

# Eliminar namespace manualmente
kubectl delete namespace zplay-vanilla

# Ver PVCs
kubectl get pvc -A | grep zplay

# Ver consumo de recursos
kubectl top pods -n zplay-vanilla
```

---

## Resumen de cambios pendientes en código

| Archivo | Cambio |
|---------|--------|
| `templates/volume.yaml` | storageClassName: nfs-shared |
| `templates/deployment.yaml` | nodeSelector dinámico |
| `templates/ingress.yaml` | entryPoints fijos (terraria1, terraria2, etc) |
| `deploy.go` | Menú de selección de nodo |
| `deploy.go` | Validación de memoria según nodo |
| `terraria.go` | Mapeo puerto → entrypoint |
| `config.go` | Añadir NodeSelector a ServerInfo |

---

## Orden de implementación recomendado

1. ⬜ Configurar Traefik con los 4 puertos
2. ⬜ Modificar `volume.yaml` → `nfs-shared`
3. ⬜ Compilar y probar deploy básico
4. ⬜ Añadir selección de nodo
5. ⬜ Añadir validaciones
6. ⬜ Implementar start/stop
7. ⬜ Implementar backups
8. ⬜ Implementar Minecraft (Phase 3)

# ZPlay - Descripción del Proyecto

## Resumen

ZPlay es una CLI en Go para desplegar, gestionar y monitorizar servidores de juegos sobre Kubernetes (k3s). El proyecto prioriza una experiencia simple para operación diaria (menú interactivo) y automatización (subcomandos con flags), encapsulando la complejidad de Kubernetes detrás de flujos consistentes.

Estado actual (20 de febrero de 2026):

- Fase 0 (fundación): completada.
- Fase 1 (robustez): completada.
- Fase 2 (variantes Terraria, incluyendo tModLoader): completada.
- Fase 3 (backup y restore): completada.
- Fase 4 (usabilidad): completada.

---

## Arquitectura

### Estructura del código

```text
zplay/
├── cmd/zplay/main.go              # Entrada CLI (modo interactivo + subcomandos)
├── internal/
│   ├── cli/
│   │   ├── cli.go                 # Menú principal (bubbletea)
│   │   ├── deploy.go              # Deploy interactivo
│   │   ├── list.go                # List interactivo + reconciliación
│   │   ├── startstop.go           # Start/Stop interactivo
│   │   ├── status.go              # Status detallado
│   │   ├── backup.go              # Backup interactivo
│   │   ├── restore.go             # Restore interactivo
│   │   ├── delete.go              # Delete interactivo
│   │   ├── console.go             # Attach a consola
│   │   ├── logs.go                # Logs
│   │   └── noninteractive.go      # Flujos directos para subcomandos
│   ├── config/
│   │   └── config.go              # Configuración + estado + reconciliación
│   ├── games/
│   │   ├── game.go                # Interfaz Game + registry
│   │   └── terraria/
│   │       ├── terraria.go        # Implementación Terraria (vanilla/tmodloader)
│   │       └── templates/         # Manifiestos K8s embed (deploy/backup/restore)
│   └── k8s/
│       └── client.go              # Wrapper de kubectl
├── docs/
│   ├── description.md
│   └── roadmap.md
├── Makefile
├── VERSION
└── go.mod
```

### Patrón de extensibilidad: Game Registry

Cada juego implementa `Game` y se registra con `init()`. El core de CLI y estado no necesita cambios para añadir operaciones comunes de un juego nuevo.

```go
type Game interface {
    Name() string
    DisplayName() string
    DefaultPort() int
    Validate(cfg *ServerConfig) error
    RenderManifests(cfg *ServerConfig) ([]string, error)
    RenderBackupJob(cfg *ServerConfig) (string, error)
    RenderRestoreJob(cfg *ServerConfig) (string, error)
    GetDeploymentName(serverName string) string
    GetNamespace(serverName string) string
}
```

### Modos de ejecución

1. Sin argumentos (`zplay`): menú interactivo.
2. Con subcomandos (`zplay deploy ...`, `zplay list --json`, etc.): modo directo no interactivo.

---

## Flujo operativo

### Deploy

1. Construcción y validación de `ServerConfig`.
2. Render de manifiestos por juego.
3. `kubectl apply` de todos los recursos.
4. Espera de readiness.
5. Persistencia en `~/.zplay/servers.yaml`.

### Reconciliación de estado

`RunList` (modo interactivo) descubre servidores reales del clúster (`app=zplay`) y compara contra estado local:

- `added`: están en clúster pero no en `servers.yaml` (opción de adopción).
- `orphaned`: están en `servers.yaml` pero ya no existen en clúster (opción de limpieza).

Esto evita depender ciegamente de `servers.yaml` como única fuente de verdad.

### Status detallado

`RunStatus` combina información de estado local y consultas de Kubernetes para mostrar:

- Estado runtime (Running/Stopped/Unknown)
- Nodo y uptime
- CPU/memoria usada (si `metrics-server` está disponible)
- Requests/limits
- PVC y storage class
- Estado de auto-backup y último backup

Si un dato no está disponible, se muestra `N/A` sin fallar el comando.

---

## Interacción con Kubernetes

ZPlay no usa `client-go`; opera mediante un wrapper de `kubectl`.

Operaciones principales del cliente K8s:

- Aplicación y borrado de recursos: `Apply`, `ApplyAll`, `DeleteNamespace`
- Estado y ciclo de vida: `WaitForReady`, `GetPodStatus`, `GetReplicas`, `ScaleDeployment`
- Operación runtime: `AttachConsole`, `Logs`, `Exec`
- Backup/restore: `RunBackupJob`, `RunRestoreJob`
- Reconciliación/inspección: `DiscoverServers`, `GetDeployments`
- Datos de status: `GetPodNodeName`, `GetPodStartTime`, `GetPodTop`, `GetDeploymentResources`, `GetPVCInfo`, `HasBackupCronJob`, `GetLastBackupTimestamp`

---

## Infraestructura ZCloud

### Nodos del clúster

| Nodo | RAM | Arquitectura | Rol |
|------|-----|-------------|-----|
| `oracle1` | 24 GB | ARM64 | Worker principal |
| `oracle2` | 24 GB | ARM64 | Worker principal |
| `raspberry` | 8 GB | ARM64 | Cargas ligeras |
| `lake` | 16 GB | AMD64 | Control plane / x86 |

Regla importante: tModLoader requiere `lake` (x86).

### Storage

- `nfs-shared`: almacenamiento persistente principal para worlds/configs.
- Backups en `/mnt/das/zplay-backups`.

### Red

- Dominio: `play.zyrak.cloud`
- Ingress TCP con Traefik (`IngressRouteTCP`)
- Mapeo de puertos:
  - Terraria: `7777 -> terraria1`, `7778 -> terraria2`
  - Minecraft (futuro): `25565 -> minecraft1`, `25566 -> minecraft2`

---

## Juegos soportados

### Terraria

- Variantes: `vanilla`, `tmodloader`
- Recursos por servidor: namespace, PVC, deployment, service, ingress TCP
- Auto-backup opcional con `CronJob`
- Backup manual y restore con jobs temporales

### Minecraft

- Planificado, no implementado todavía.

---

## Configuración del usuario

### Archivo de configuración

`~/.zplay/config.yaml`

```yaml
domain: play.zyrak.cloud
kubeconfig: ~/.zcloud/kubeconfig
node_selector: ""
data_path: ~/.zplay
```

### Estado local

`~/.zplay/servers.yaml`

```yaml
servers:
  - name: vanilla
    game: terraria
    namespace: zplay-vanilla
    variant: vanilla
    auto_backup: true
    node: oracle1
    port: 7777
    memory: 4Gi
    max_players: 8
    created_at: "2025-01-15T10:30:00Z"
```

---

## Seguridad

### Passwords

Cuando se define contraseña, ZPlay crea un `Secret` y el deployment referencia `valueFrom.secretKeyRef`.

### Permisos de clúster

ZPlay hereda permisos del kubeconfig activo. No implementa RBAC propio a nivel de aplicación.

### Aislamiento

Cada servidor vive en su propio namespace (`zplay-{name}`), facilitando aislamiento y borrado completo.

---

## Limitaciones actuales

1. Minecraft aún no está implementado.
2. La reconciliación automática solo existe en `list` interactivo (`--sync` no implementado todavía).
3. La adopción de servidores descubiertos infiere algunos campos locales (por ejemplo, puerto/memoria por defecto).
4. Sin control de acceso multiusuario dentro de la CLI (depende de kubeconfig/RBAC del clúster).
5. Métricas CPU/memoria dependen de `metrics-server`; sin él se muestra `N/A`.

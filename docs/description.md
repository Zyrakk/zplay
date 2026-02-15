# ZPlay - Descripción del Proyecto

## Resumen

ZPlay es una herramienta CLI interactiva escrita en Go que permite desplegar, gestionar y monitorizar servidores de juegos sobre un clúster Kubernetes (k3s). El objetivo es abstraer toda la complejidad de Kubernetes detrás de un menú interactivo donde el usuario selecciona opciones y el sistema se encarga de generar manifiestos, aplicarlos al clúster y gestionar el ciclo de vida completo del servidor.

El proyecto está diseñado específicamente para la infraestructura ZCloud, un clúster k3s personal con nodos ARM64, almacenamiento NFS compartido y Traefik como ingress controller.

---

## Arquitectura

### Estructura del código

```
zplay/
├── cmd/zplay/main.go              # Punto de entrada
├── internal/
│   ├── cli/                        # Interfaz interactiva
│   │   ├── cli.go                  # Menú principal (bubbletea)
│   │   ├── deploy.go               # Flujo de deploy
│   │   ├── list.go                 # Listado de servidores
│   │   ├── delete.go               # Eliminación con confirmación
│   │   ├── console.go              # Attach a consola del servidor
│   │   └── logs.go                 # Visualización de logs
│   ├── config/
│   │   └── config.go               # Configuración y persistencia de estado
│   ├── games/
│   │   ├── game.go                 # Interfaz Game + registro global
│   │   └── terraria/
│   │       ├── terraria.go         # Implementación de Terraria
│   │       └── templates/          # Manifiestos K8s (embed)
│   │           ├── namespace.yaml
│   │           ├── volume.yaml
│   │           ├── deployment.yaml
│   │           ├── service.yaml
│   │           └── ingress.yaml
│   └── k8s/
│       └── client.go               # Wrapper sobre kubectl
├── scripts/
│   └── bump-version.sh             # Gestión de versiones
├── Makefile                        # Build, install, dev, test
├── VERSION                         # Versión actual (0.1.0)
└── go.mod                          # Dependencias Go
```

### Patrón de extensibilidad: Game Registry

El sistema usa un patrón de registro para los juegos. Cada juego implementa la interfaz `Game`:

```go
type Game interface {
    Name() string
    DisplayName() string
    DefaultPort() int
    Validate(cfg *ServerConfig) error
    RenderManifests(cfg *ServerConfig) ([]string, error)
    GetDeploymentName(serverName string) string
    GetNamespace(serverName string) string
}
```

Los juegos se auto-registran mediante `init()` en su paquete, y se importan con blank import en `cli.go`:

```go
_ "github.com/Zyrakk/zplay/internal/games/terraria"
```

Para añadir un juego nuevo (ej: Minecraft), basta con crear el paquete `internal/games/minecraft/` con su implementación y templates, e importarlo en `cli.go`. No hace falta modificar ningún otro fichero del core.

### Flujo de deploy

1. El usuario selecciona el juego y configura parámetros (nombre, memoria, puerto, etc.)
2. Se valida la configuración con `game.Validate()`
3. Se renderizan los templates YAML con `text/template` y los datos del `ServerConfig`
4. Se aplican los manifiestos al clúster con `kubectl apply`
5. Se espera a que el deployment esté ready
6. Se guarda el estado en `~/.zplay/servers.yaml`

### Interacción con Kubernetes

ZPlay NO usa `client-go`. Toda la interacción con Kubernetes se hace a través de un wrapper sobre `kubectl` que ejecuta comandos como subprocesos. Esto simplifica enormemente las dependencias y el código, a cambio de requerir que kubectl esté instalado y configurado.

Operaciones disponibles en el cliente K8s:
- `Apply` / `ApplyAll` — Aplicar manifiestos
- `DeleteNamespace` — Eliminar servidor completo
- `GetPodStatus` — Estado del pod
- `WaitForReady` — Esperar a que el deployment esté disponible
- `AttachConsole` — Attach interactivo (stdin/stdout)
- `Logs` — Ver/seguir logs
- `Exec` — Ejecutar comandos en el pod
- `GetDeployments` — Listar deployments por label
- `IsConnected` — Verificar conectividad al clúster

### Persistencia de estado

El estado de los servidores se mantiene en un fichero YAML local (`~/.zplay/servers.yaml`). Cada servidor registra: nombre, juego, puerto, memoria, max players y fecha de creación.

**Limitación conocida**: el fichero local es la única fuente de verdad. Si se pierde o se ejecuta zplay desde otra máquina, se pierde la referencia a los servidores (aunque siguen existiendo en el clúster). Una mejora futura sería derivar el estado del clúster usando los labels `app: zplay` que ya se aplican a todos los recursos.

---

## Infraestructura ZCloud

### Nodos del clúster

| Nodo | RAM | Arquitectura | Almacenamiento | Rol |
|------|-----|-------------|----------------|-----|
| **oracle1** | 24 GB | ARM64 | 5 TB en /mnt/das | Worker — juegos pesados |
| **oracle2** | 24 GB | ARM64 | 5 TB en /mnt/das | Worker — juegos pesados |
| **raspberry** | 8 GB | ARM64 | 4 TB en /mnt/local | Worker — servidores ligeros |
| **lake** | 16 GB | AMD64 (x86) | — | Control plane — reservado para tModLoader (único nodo x86) |

**Restricciones de scheduling**:
- `lake` ejecuta el control plane. Solo debe usarse para cargas que requieran x86, concretamente tModLoader.
- `raspberry` tiene solo 8 GB y debe limitarse a servidores vanilla ligeros (Terraria sin mods, max 4 GB).
- `oracle1` y `oracle2` son los nodos principales para juegos vanilla y cargas ARM64.

### Almacenamiento

| StorageClass | Capacidad | Uso |
|-------------|-----------|-----|
| `nfs-shared` | 5 TB (LVM) | Datos de juego (worlds, configs) — accesible desde cualquier nodo |
| `nfs-nvme` | 500 GB | Reservado (VictoriaMetrics, Wazuh) — NO usar |
| `local-path` | Variable | NO usar para game servers (no permite migración entre nodos) |

**Importante**: Los templates de Terraria ya usan `nfs-shared` para persistencia compartida entre nodos.

**Backups**: El destino de backups es `/mnt/das` en oracle1 u oracle2, separado del storage principal NFS para que las backups sobrevivan a un fallo del storage compartido.

### Red e Ingress

**Ingress controller**: Traefik (incluido con k3s).

ZPlay usa `IngressRouteTCP` (CRD de Traefik) para exponer los puertos de juego como TCP puro. Cada servidor necesita un entrypoint configurado en Traefik.

**Dominio**: `play.zyrak.cloud`

**Puertos asignados**:

| Puerto | Juego | Protocolo |
|--------|-------|-----------|
| 7777 | Terraria Server 1 | TCP |
| 7778 | Terraria Server 2 | TCP |
| 25565 | Minecraft Server 1 | TCP |
| 25566 | Minecraft Server 2 | TCP |

**Configuración requerida en Traefik**: Los entrypoints deben configurarse manualmente en el HelmChartConfig de Traefik en el control plane (`/var/lib/rancher/k3s/server/manifests/traefik-config.yaml`). Traefik no permite añadir entrypoints dinámicamente; requiere reinicio.

La estrategia óptima es definir un set fijo de entrypoints (terraria1, terraria2, minecraft1, minecraft2) y que el código mapee cada puerto a su entrypoint correspondiente, validando que el puerto solicitado tiene un entrypoint configurado.

---

## Juegos soportados

### Terraria

**Estado**: Implementación base completada.

**Imagen Docker**: `hexlo/terraria-server-docker:latest`
Nota: esta imagen se eligió por soporte multi-arch (amd64 + arm64), necesario para desplegar tanto en nodos Oracle ARM64 como en x86.

**Parámetros configurables**:
- Nombre del servidor
- Tamaño del mundo (small/medium/large)
- Máximo de jugadores (1-255)
- Password (opcional)
- Puerto (default: 7777)
- Memoria (default: 4Gi request, 8Gi limit)

**Variantes planificadas**:
- **Vanilla** — La implementación actual, servidor base sin modificaciones.
- **tModLoader** — Servidor con soporte de mods. Se forzará al nodo `lake` (único x86) y usará imagen específica de tModLoader en Fase 2.

**Recursos K8s generados por servidor**:
- Namespace (`zplay-{nombre}`)
- PersistentVolumeClaim (10 Gi)
- Deployment (1 réplica, con probes)
- Service (ClusterIP, puerto 7777)
- IngressRouteTCP (Traefik)

### Minecraft (planificado)

**Imagen Docker**: `itzg/minecraft-server` — imagen muy completa con soporte para múltiples tipos de servidor.

**Variantes planificadas**:
- **Vanilla** — Servidor base de Minecraft.
- **Paper** — Servidor optimizado con soporte de plugins. Es el tipo por defecto recomendado.
- **Forge** — Servidor con soporte de mods (lado servidor y cliente).

**Parámetros específicos**:
- Tipo de servidor (VANILLA/PAPER/FORGE)
- Versión de Minecraft
- MOTD (mensaje del servidor)
- Lista de operadores
- RCON (consola remota, siempre habilitado)
- Whitelist
- Plugins/Mods

---

## Configuración del usuario

### Fichero de configuración

Ubicación: `~/.zplay/config.yaml`

```yaml
domain: play.zyrak.cloud
kubeconfig: ~/.zcloud/kubeconfig
node_selector: ""           # Nodo por defecto (vacío = scheduler decide)
data_path: ~/.zplay          # Directorio de datos locales
```

### Fichero de estado

Ubicación: `~/.zplay/servers.yaml`

```yaml
servers:
  - name: vanilla
    game: terraria
    port: 7777
    memory: 4Gi
    max_players: 8
    created_at: "2025-01-15T10:30:00Z"
```

### Requisitos del sistema

- Go 1.22+ (solo para compilación)
- `kubectl` instalado y en PATH
- Acceso al clúster k3s (kubeconfig configurado)
- Traefik con los entrypoints TCP configurados

---

## Dependencias Go

| Paquete | Uso |
|---------|-----|
| `charmbracelet/bubbletea` | Framework TUI para el menú interactivo |
| `charmbracelet/lipgloss` | Estilos y colores en terminal |
| `gopkg.in/yaml.v3` | Serialización de configuración y estado |

---

## Consideraciones de seguridad

### Passwords

Actualmente las contraseñas se inyectan como variables de entorno en plaintext dentro del manifiesto de Deployment. Esto significa que cualquier persona con acceso al namespace puede ver la contraseña haciendo `kubectl describe deployment`.

La solución planificada es crear un `Secret` de Kubernetes y referenciar la contraseña con `valueFrom.secretKeyRef` en el Deployment.

### Acceso al clúster

ZPlay hereda los permisos del kubeconfig configurado. No implementa ningún control de acceso propio — se asume que el usuario tiene permisos completos sobre el clúster (uso personal/homelab).

### Namespaces

Cada servidor se despliega en su propio namespace (`zplay-{nombre}`), lo que proporciona aislamiento de recursos y simplifica la limpieza (eliminar un servidor = eliminar el namespace completo).

---

## Limitaciones actuales

1. **Passwords en plaintext**: No se usan Secrets de Kubernetes para password.
2. **Sin validación de recursos en clúster**: No se verifica capacidad real de RAM/CPU antes del deploy.
3. **Sin validación de puertos en uso real del clúster**: Solo hay estado local, no reconciliación completa.
4. **Sin start/stop**: Solo se puede crear o eliminar servidores, no pausarlos/reanudarlos.
5. **Sin backups**: No hay mecanismo de backup/restore integrado.
6. **Sin modo no interactivo**: No se pueden ejecutar comandos directamente con flags.
7. **Estado local como fuente primaria**: Si se pierde `servers.yaml`, se pierde la referencia local a servidores existentes.
8. **Sin Minecraft**: Solo Terraria está implementado actualmente.
9. **Sin template de tModLoader aún**: Existe preparación de variante y restricciones de nodo, pero falta implementación completa.
10. **Sin menú de dificultad todavía**: Se usa dificultad por defecto (classic) salvo ajuste manual futuro.

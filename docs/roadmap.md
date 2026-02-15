# ZPlay - Roadmap

## Estado actual (v0.1.0)

Lo que ya está implementado y funcional a nivel de código:

- Estructura del proyecto con patrón extensible (Game Registry)
- CLI interactivo con bubbletea (menú principal con navegación)
- Comandos: deploy, list, delete, console, logs
- Implementación de Terraria vanilla con templates K8s
- Cliente K8s wrapper sobre kubectl
- Persistencia de configuración y estado de servidores
- Sistema de versionado y Makefile completo

**Lo que NO funciona aún en producción**: los templates tienen configuraciones incorrectas para la infraestructura real (storageClass, entrypoints, etc.) y faltan validaciones críticas. El código compila pero no se ha probado contra el clúster.

---

## Fase 0 — Fundación (hacer funcionar lo que hay)

**Objetivo**: Conseguir que el código existente compile, se conecte al clúster y despliegue un servidor de Terraria vanilla que sea accesible desde fuera.

### 0.1 Compilación y dependencias

- [ ] Ejecutar `go mod tidy` para sincronizar dependencias
- [ ] Verificar que compila limpio: `go build ./cmd/zplay`
- [ ] Eliminar `charmbracelet/huh` de go.mod si no se va a usar inmediatamente, o empezar a usarlo en los formularios
- [ ] Probar ejecución local: `make dev`

### 0.2 Corregir templates para la infraestructura real

#### volume.yaml
- [ ] Cambiar `storageClassName` de `local-path` a `nfs-shared`
- [ ] Evaluar si 10Gi es suficiente para Terraria (los worlds grandes pueden crecer)

#### ingress.yaml
- [ ] Cambiar de entrypoints dinámicos (`zplay-{{.Port}}`) a entrypoints fijos
- [ ] Crear función de mapeo puerto→entrypoint en `terraria.go`:
  ```
  7777 → terraria1
  7778 → terraria2
  ```
- [ ] Añadir el entrypoint como campo en ServerConfig y pasarlo al template
- [ ] Actualizar el template para usar `{{.Entrypoint}}` en vez de `zplay-{{.Port}}`

### 0.3 Configurar Traefik en el clúster

- [ ] Editar `/var/lib/rancher/k3s/server/manifests/traefik-config.yaml` en n150
- [ ] Añadir los 4 entrypoints TCP: terraria1 (7777), terraria2 (7778), minecraft1 (25565), minecraft2 (25566)
- [ ] Verificar que Traefik recarga la configuración
- [ ] Comprobar que los puertos están expuestos: `kubectl get svc -n kube-system traefik`

### 0.4 Primer deploy real

- [ ] Ejecutar `zplay` y desplegar un servidor de Terraria
- [ ] Verificar que se crea el namespace, PVC, deployment, service e ingress
- [ ] Verificar que el pod arranca y pasa los probes
- [ ] Conectar al servidor desde un cliente de Terraria en `play.zyrak.cloud:7777`
- [ ] Probar los comandos: list (verificar estado Running), logs, console, delete

### 0.5 Fixes post-test

- [ ] Documentar cualquier problema encontrado y corregir
- [ ] Verificar que delete limpia correctamente todos los recursos
- [ ] Confirmar que el PVC con nfs-shared persiste datos entre reinicios del pod

**Criterio de completado**: Un servidor de Terraria vanilla desplegado, accesible, con datos persistentes, gestionable con todos los comandos del menú.

---

## Fase 1 — Robustez (validaciones y mejoras críticas)

**Objetivo**: Hacer el deploy seguro y predecible. Que no se pueda romper nada por error del usuario.

### 1.1 Selección de nodo

- [ ] Añadir menú de selección de nodo en `deploy.go`:
  ```
  Select node:
    1) oracle1 (24GB RAM) - recommended
    2) oracle2 (24GB RAM)
    3) raspberry (8GB RAM) - light servers only
    4) Auto (scheduler decides)
  ```
- [ ] Pasar el nodo seleccionado como `NodeSelector` al `ServerConfig`
- [ ] Guardar el nodo en `ServerInfo` para mostrarlo en list

### 1.2 Validaciones

- [ ] Validar que el puerto solicitado tiene un entrypoint configurado (lista fija de puertos permitidos por juego)
- [ ] Validar que el puerto no está ya en uso por otro servidor en el estado
- [ ] Validar formato de memoria (debe terminar en Gi o Mi, con valor numérico)
- [ ] Si se selecciona raspberry, limitar memoria máxima a 4Gi
- [ ] Validar que el nombre del servidor es DNS-compatible (lowercase, sin espacios, sin caracteres especiales)

### 1.3 Secrets para passwords

- [ ] Crear template `secret.yaml` en los templates de Terraria
- [ ] Actualizar `deployment.yaml` para referenciar el secret con `valueFrom.secretKeyRef`
- [ ] Condicionar la creación del secret solo cuando hay password configurado
- [ ] Añadir `secret.yaml` a la lista de templates en `terraria.go`

### 1.4 Start/Stop de servidores

- [ ] Añadir método `ScaleDeployment(namespace, deployment, replicas)` al cliente K8s
- [ ] Crear `internal/cli/startstop.go` con flujo interactivo:
  - Listar servidores con su estado actual
  - Seleccionar servidor
  - Si está running → ofrecer stop (scale a 0)
  - Si está stopped → ofrecer start (scale a 1)
- [ ] Añadir "Start/Stop server" al menú principal
- [ ] Actualizar `list.go` para distinguir entre "Stopped" (deployment con 0 réplicas) y "Not Found"

### 1.5 Dificultad de Terraria

- [ ] Añadir menú de selección de dificultad en deploy:
  ```
  Difficulty:
    1) Classic (Normal)
    2) Expert
    3) Master
    4) Journey
  ```
- [ ] Mapear la selección al valor numérico (0=Classic, 1=Expert, 2=Master, 3=Journey)
- [ ] Pasar el valor al template via `ServerConfig.Difficulty`
- [ ] Mostrar la dificultad en el resumen pre-deploy

**Criterio de completado**: Deploy seguro con validaciones, passwords en secrets, start/stop funcional, dificultad configurable.

---

## Fase 2 — Terraria con mods (tModLoader)

**Objetivo**: Soportar servidores de Terraria con mods usando tModLoader como variante del juego.

### 2.1 Variante tModLoader

El enfoque es tratar tModLoader como una variante dentro de la implementación de Terraria, no como un juego separado. Comparte la mayoría de la configuración pero usa una imagen Docker diferente y tiene parámetros adicionales.

- [ ] Añadir campo `Variant` al `ServerConfig` (valores: "vanilla", "tmodloader")
- [ ] Modificar el menú de deploy de Terraria para preguntar la variante:
  ```
  Server type:
    1) Vanilla
    2) tModLoader (mods)
  ```
- [ ] Crear templates alternativos o condicionales para tModLoader

### 2.2 Templates tModLoader

La imagen de tModLoader es diferente y tiene variables de entorno distintas.

- [ ] Investigar y documentar la imagen `jacobsmile/tmodloader:latest`:
  - Variables de entorno disponibles
  - Puertos requeridos
  - Rutas de datos (worlds, mods)
  - Requisitos de recursos (CPU, RAM mínima)
- [ ] Crear template `deployment-tmodloader.yaml` o condicionar el template existente:
  ```yaml
  image: {{ if eq .Variant "tmodloader" }}jacobsmile/tmodloader:latest{{ else }}passivelemon/terraria-docker:terraria-latest{{ end }}
  ```
- [ ] Ajustar volumeMounts según la imagen (la ruta de datos puede ser diferente)
- [ ] Ajustar los probes (tModLoader puede tardar más en arrancar)

### 2.3 Gestión de mods

- [ ] Definir la estrategia de instalación de mods:
  - **Opción A**: El usuario monta los mods manualmente en el PVC
  - **Opción B**: Configurar mods por nombre/ID en el deploy y que la imagen los descargue
  - **Opción C**: Comando separado `zplay mods add/remove` post-deploy
- [ ] Implementar la estrategia elegida
- [ ] Documentar cómo encontrar IDs de mods compatibles

### 2.4 Validaciones específicas de tModLoader

- [ ] Aumentar la memoria mínima recomendada para tModLoader (los mods consumen más)
- [ ] Advertir si se selecciona raspberry como nodo para tModLoader
- [ ] Validar que la versión de tModLoader es compatible con los mods seleccionados (si aplica)

### 2.5 Testing de Terraria completo

- [ ] Desplegar servidor vanilla y verificar que sigue funcionando
- [ ] Desplegar servidor tModLoader sin mods
- [ ] Desplegar servidor tModLoader con al menos un mod
- [ ] Verificar persistencia de datos (worlds y mods) tras reinicio
- [ ] Verificar que list muestra correctamente la variante
- [ ] Probar start/stop con ambas variantes

**Criterio de completado**: Terraria vanilla y tModLoader desplegables, gestionables y estables. Un usuario puede elegir entre vanilla y mods al desplegar.

---

## Fase 3 — Backups

**Objetivo**: Proteger los datos de los servidores con backups manuales y automáticos.

### 3.1 Backup manual

- [ ] Crear comando `zplay backup` (o añadir al menú como "Backup server")
- [ ] Flujo:
  1. Seleccionar servidor
  2. Ejecutar backup: crear un Job temporal que monta el PVC del servidor y copia los datos comprimidos a `/mnt/das/zplay-backups/`
  3. Nombrar el backup como `{nombre}-{fecha}.tar.gz`
  4. Mostrar confirmación con la ruta del backup
- [ ] Implementar el Job de backup como template K8s:
  ```yaml
  apiVersion: batch/v1
  kind: Job
  metadata:
    name: {{.Name}}-backup-manual
    namespace: zplay-{{.Name}}
  spec:
    template:
      spec:
        containers:
          - name: backup
            image: alpine
            command: ["tar", "-czf", "/backup/{{.Name}}-{{.Timestamp}}.tar.gz", "/data"]
            volumeMounts:
              - name: data
                mountPath: /data
                readOnly: true
              - name: backup
                mountPath: /backup
        restartPolicy: Never
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
- [ ] Esperar a que el Job complete y limpiar el Job después

### 3.2 Backup automático

- [ ] Crear template `cronjob-backup.yaml`:
  - Ejecución diaria a las 4:00 AM
  - Mismo mecanismo que el backup manual
  - Retención: mantener últimos 7 backups (script de limpieza en el container)
- [ ] Hacer el CronJob opcional en el deploy (preguntar si activar auto-backup)
- [ ] Incluir el CronJob en los manifiestos rendidos si se activa

### 3.3 Restauración

- [ ] Crear comando `zplay restore` (o "Restore backup" en el menú)
- [ ] Flujo:
  1. Seleccionar servidor
  2. Listar backups disponibles (ls en /mnt/das/zplay-backups/ filtrando por nombre)
  3. Seleccionar backup
  4. Confirmar (warning: sobrescribirá datos actuales)
  5. Escalar deployment a 0 (stop)
  6. Ejecutar Job de restauración (extraer tar al PVC)
  7. Escalar deployment a 1 (start)
- [ ] Validar que el backup es del mismo juego/variante que el servidor destino

**Criterio de completado**: Backup manual y automático funcionales. Restauración probada con al menos un servidor de Terraria.

---

## Fase 4 — Mejoras de usabilidad

**Objetivo**: Pulir la experiencia de usuario y añadir funcionalidades que faciliten el día a día.

### 4.1 Reconciliación de estado con el clúster

- [ ] Implementar función que lea los recursos del clúster con label `app: zplay` y reconstruya el estado
- [ ] Usar al inicio de `list.go` para mostrar siempre el estado real
- [ ] Si hay discrepancias entre `servers.yaml` y el clúster, mostrar warning y ofrecer sincronizar
- [ ] Permitir adoptar servidores huérfanos (existen en clúster pero no en servers.yaml)

### 4.2 Modo no interactivo (CLI flags)

- [ ] Añadir soporte para subcomandos directos:
  ```
  zplay deploy --game terraria --name vanilla --memory 4Gi --node oracle1 --port 7777
  zplay deploy --game terraria --name modded --variant tmodloader --node oracle2 --port 7778
  zplay list [--json]
  zplay delete <nombre> --yes
  zplay stop <nombre>
  zplay start <nombre>
  zplay backup <nombre>
  zplay restore <nombre> --backup <fichero>
  zplay status <nombre>
  ```
- [ ] Si se ejecuta `zplay` sin argumentos, mantener el menú interactivo
- [ ] Evaluar usar `cobra` o mantener el parsing manual con `os.Args`

### 4.3 Status detallado

- [ ] Crear comando `status` que muestre información completa de un servidor:
  ```
  Server:      vanilla
  Game:        Terraria (vanilla)
  Status:      Running
  Node:        oracle1
  Memory:      3.2Gi / 4Gi (request) / 8Gi (limit)
  CPU:         0.3 / 2 cores
  Uptime:      3d 14h 22m
  Port:        play.zyrak.cloud:7777
  Created:     2025-01-15 10:30:00
  PVC:         10Gi (nfs-shared)
  Auto-backup: Enabled (daily 4:00 AM)
  Last backup: 2025-01-18 04:00:12
  ```
- [ ] Obtener métricas de `kubectl top pod` si metrics-server está disponible

### 4.4 Mejoras visuales del CLI

- [ ] Usar `charmbracelet/huh` para los formularios de deploy (reemplazar los prompts manuales con bufio.Reader)
- [ ] Añadir spinners/progress bars para operaciones largas (deploy, backup, restore)
- [ ] Mejorar la tabla de list con formato más limpio

**Criterio de completado**: CLI robusto tanto en modo interactivo como en modo flag. Estado siempre sincronizado con el clúster real.

---

## Fase 5 — Minecraft

**Objetivo**: Añadir soporte completo para Minecraft con tres variantes: Vanilla, Paper (plugins) y Forge (mods).

### 5.1 Implementación base de Minecraft

- [ ] Crear directorio `internal/games/minecraft/`
- [ ] Implementar la interfaz `Game` para Minecraft:
  ```go
  type Minecraft struct{}
  func (m *Minecraft) Name() string        { return "minecraft" }
  func (m *Minecraft) DisplayName() string { return "Minecraft" }
  func (m *Minecraft) DefaultPort() int    { return 25565 }
  ```
- [ ] Definir los campos específicos en `ServerConfig`:
  - `ServerType` (VANILLA, PAPER, FORGE)
  - `Version` (versión de Minecraft)
  - `JavaMemory` (diferente de Memory del pod; es el heap de Java)
  - `MOTD` (mensaje del servidor)
  - `Ops` (lista de operadores)
  - `RCONPassword` (generado automáticamente)
- [ ] Registrar con `games.Register(&Minecraft{})` en `init()`
- [ ] Importar en `cli.go`: `_ "github.com/Zyrakk/zplay/internal/games/minecraft"`

### 5.2 Templates K8s para Minecraft

Usar la imagen `itzg/minecraft-server` que soporta todas las variantes via variables de entorno.

#### namespace.yaml
- [ ] Mismo patrón que Terraria: `zplay-{nombre}`

#### volume.yaml
- [ ] StorageClass: `nfs-shared`
- [ ] Evaluar tamaño: Minecraft worlds pueden ser mucho más grandes. Considerar 20-50Gi según variante.

#### deployment.yaml
- [ ] Imagen: `itzg/minecraft-server`
- [ ] Variables de entorno clave:
  ```yaml
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
  - name: ENABLE_RCON
    value: "true"
  - name: RCON_PASSWORD
    valueFrom:
      secretKeyRef:
        name: {{.Name}}-secret
        key: rcon-password
  ```
- [ ] Puertos: 25565 (game) y 25575 (RCON)
- [ ] Probes: TCP en puerto 25565 (initialDelay más alto que Terraria, Minecraft tarda más)
- [ ] Recursos: request 4Gi/1cpu, limit 8Gi/2cpu (configurable)

#### service.yaml
- [ ] Puertos: game (25565) y rcon (25575)

#### secret.yaml
- [ ] RCON password (generado automáticamente)
- [ ] Server password (si aplica)

#### ingress.yaml
- [ ] Mismo patrón de IngressRouteTCP con entrypoints fijos (minecraft1, minecraft2)

### 5.3 Flujo de deploy de Minecraft

- [ ] Menú de deploy específico:
  ```
  Server type:
    1) Paper (recommended - optimized, plugins)
    2) Vanilla
    3) Forge (mods)

  Minecraft version [latest]:
  Server name:
  MOTD [A Minecraft Server]:
  Max players [20]:
  Memory [4Gi]:
  Node [oracle1]:
  Port [25565]:
  Password (optional):
  Operators (comma-separated usernames):
  ```
- [ ] Validaciones:
  - Paper/Forge: mínimo 4Gi de memoria recomendado
  - Forge: no recomendar raspberry
  - Versión: validar formato (ej: 1.20.4) o aceptar "latest"
  - Operadores: validar formato de usernames de Minecraft

### 5.4 RCON Console

La imagen itzg/minecraft-server incluye `rcon-cli` que permite ejecutar comandos sin hacer attach al proceso.

- [ ] Modificar `console.go` para detectar si el juego es Minecraft
- [ ] Si es Minecraft, ofrecer dos opciones:
  ```
  Console type:
    1) RCON (recommended - send commands)
    2) Container attach (raw stdout/stdin)
  ```
- [ ] Para RCON, usar `kubectl exec` para ejecutar `rcon-cli` dentro del pod:
  ```go
  client.Exec(namespace, deployment, []string{"rcon-cli"})
  ```

### 5.5 Minecraft Vanilla — Testing

- [ ] Desplegar servidor Vanilla
- [ ] Verificar que arranca correctamente (puede tardar 2-3 min)
- [ ] Conectar con cliente de Minecraft
- [ ] Verificar RCON funcional
- [ ] Verificar persistencia del world tras reinicio
- [ ] Probar stop/start
- [ ] Probar backup/restore

**Criterio de completado**: Minecraft Vanilla desplegable y funcional con RCON, persistencia y backups.

---

## Fase 6 — Minecraft con plugins y mods

**Objetivo**: Soportar Paper (plugins) y Forge (mods) como variantes de Minecraft.

### 6.1 Paper (plugins)

Paper es el tipo más común y la imagen itzg lo soporta nativamente con `TYPE=PAPER`.

- [ ] Verificar que el deploy con `TYPE=PAPER` funciona correctamente
- [ ] Investigar gestión de plugins:
  - La imagen itzg soporta descarga automática de plugins con `SPIGET_RESOURCES` o `MODRINTH_PROJECTS`
  - Alternativa: montar plugins manualmente en el PVC
- [ ] Decidir estrategia de plugins e implementar:
  - **Mínimo viable**: documentar cómo copiar plugins al PVC manualmente
  - **Avanzado**: preguntar IDs de plugins en el deploy y pasarlos como env vars
- [ ] Testear con al menos 2-3 plugins populares (EssentialsX, WorldEdit, etc.)

### 6.2 Forge (mods)

Forge requiere más configuración porque los mods deben ser compatibles entre sí y con la versión exacta de Forge.

- [ ] Verificar que el deploy con `TYPE=FORGE` funciona
- [ ] Investigar gestión de mods con la imagen itzg:
  - Soporta `FORGE_VERSION` para especificar versión de Forge
  - Puede usar `MODS` para URLs directas de descarga
  - Puede usar CurseForge API para descarga por slug
- [ ] Implementar selección de versión de Forge en el deploy
- [ ] Estrategia de mods: misma decisión que plugins de Paper
- [ ] Testear con al menos un modpack pequeño

### 6.3 Gestión de plugins/mods post-deploy

Si se implementa gestión avanzada:

- [ ] Comando `zplay plugins <servidor>` (solo Paper):
  - Listar plugins instalados
  - Añadir plugin por nombre/ID
  - Eliminar plugin
  - Reiniciar servidor tras cambios
- [ ] Comando `zplay mods <servidor>` (solo Forge/tModLoader):
  - Mismo flujo que plugins

### 6.4 Testing completo de Minecraft

- [ ] Paper sin plugins: deploy, connect, RCON
- [ ] Paper con plugins: verificar que los plugins se cargan
- [ ] Forge sin mods: deploy, connect
- [ ] Forge con mods: verificar que los mods se cargan
- [ ] Backups con cada variante
- [ ] Start/stop con cada variante
- [ ] List muestra correctamente tipo (Paper/Forge/Vanilla)

**Criterio de completado**: Las tres variantes de Minecraft son desplegables, gestionables y estables. Los plugins/mods persisten entre reinicios.

---

## Fase 7 — Producción

**Objetivo**: Estabilizar todo y preparar para uso continuado y fiable.

### 7.1 Revisión de estabilidad

- [ ] Dejar servidores corriendo durante al menos una semana y monitorizar:
  - Que no hay crashes o reinicios inesperados
  - Que los probes funcionan correctamente
  - Que el consumo de recursos es estable
  - Que los backups automáticos se ejecutan correctamente
- [ ] Ajustar los valores de probes si es necesario (initialDelay, timeout, failureThreshold)
- [ ] Ajustar los requests/limits de recursos según el uso real observado

### 7.2 Documentación final

- [ ] Actualizar README.md con:
  - Todos los juegos y variantes soportados
  - Ejemplos de uso interactivo y no interactivo
  - Guía de configuración de Traefik
  - Guía de troubleshooting
- [ ] Documentar la estructura de backups y cómo restaurar manualmente si zplay falla
- [ ] Documentar cómo añadir un juego nuevo (guía de desarrollo)

### 7.3 Limpieza de código

- [ ] Revisar TODOs y FIXMEs en el código
- [ ] Eliminar código muerto o comentado
- [ ] Asegurar que `go vet` y `golangci-lint` pasan sin warnings
- [ ] Añadir tests unitarios para las funciones críticas:
  - Validaciones de ServerConfig
  - Mapeo de puertos a entrypoints
  - Renderizado de templates (verificar YAML válido)
  - Lógica de NextPort
- [ ] Asegurar que `make build-all` genera binarios para todas las plataformas

### 7.4 Release v1.0.0

- [ ] Bump versión a 1.0.0
- [ ] Tag en git
- [ ] Build de release para linux/arm64 (la plataforma principal del clúster)
- [ ] Instalar en la máquina de gestión

**Criterio de completado**: ZPlay v1.0 instalado, con Terraria (vanilla + tModLoader) y Minecraft (Vanilla + Paper + Forge) funcionando en producción, con backups automáticos y todas las funcionalidades estables.

---

## Resumen de fases

| Fase | Nombre | Juegos | Entregable principal |
|------|--------|--------|---------------------|
| 0 | Fundación | Terraria vanilla | Primer deploy funcional en el clúster |
| 1 | Robustez | Terraria vanilla | Validaciones, secrets, start/stop, dificultad |
| 2 | tModLoader | Terraria vanilla + mods | Soporte completo de Terraria |
| 3 | Backups | Todos | Backup manual, automático y restauración |
| 4 | Usabilidad | Todos | CLI flags, status detallado, reconciliación de estado |
| 5 | Minecraft | MC Vanilla | Implementación base de Minecraft |
| 6 | MC Variantes | MC Vanilla + Paper + Forge | Plugins y mods para Minecraft |
| 7 | Producción | Todos | Estabilidad, docs, tests, release v1.0 |

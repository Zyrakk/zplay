# ZPlay - Roadmap

## Estado actual (20 de febrero de 2026)

Fases completadas:

- Fase 0 - Fundación
- Fase 1 - Robustez
- Fase 2 - Terraria variantes (vanilla + tModLoader)
- Fase 3 - Backups y restore
- Fase 4 - Usabilidad

Resultado actual:

- Menú interactivo completo para deploy/list/start-stop/status/backup/restore/delete/console/logs.
- Modo no interactivo por subcomandos con flags (`deploy`, `list`, `delete`, `start`, `stop`, `backup`, `status`).
- Reconciliación de estado local con clúster desde `list` interactivo (adoptar/limpiar).
- Status detallado por servidor con fallback a `N/A` cuando faltan métricas.

---

## Fase 5 - Minecraft

**Objetivo**: añadir soporte productivo para Minecraft con variantes y operación equivalente a Terraria.

### 5.1 Base del juego

- [ ] Crear implementación `internal/games/minecraft/` que cumpla la interfaz `Game`.
- [ ] Registrar el juego en el registry y exponerlo en deploy.
- [ ] Definir templates base (namespace, pvc, deployment, service, ingress).

### 5.2 Variantes

- [ ] Soporte de variantes iniciales: `vanilla`, `paper`, `forge`.
- [ ] Definir imagen y variables por variante.
- [ ] Validaciones por variante (memoria mínima, compatibilidad de versión).

### 5.3 Configuración de servidor

- [ ] Parámetros básicos: versión, MOTD, max players.
- [ ] Parámetros opcionales: whitelist, operadores, ajustes de Java.
- [ ] Mantener coherencia entre modo interactivo y no interactivo.

### 5.4 Operación y observabilidad

- [ ] Integrar start/stop/status para Minecraft.
- [ ] Verificar backup/restore sobre PVC de Minecraft.
- [ ] Añadir cobertura de tests para flujos clave.

### 5.5 Criterio de completado

- [ ] Deploy estable en clúster real.
- [ ] Gestión completa desde menú y subcomandos.
- [ ] Estado persistente y reconciliación funcionando con `app=zplay`.

---

## Fase 6 - Mejoras de operación

**Objetivo**: reducir fricción operativa y mejorar automatización.

- [ ] Implementar `zplay list --sync` (modo no interactivo para reconciliación).
- [ ] Añadir subcomando no interactivo para restore (`zplay restore <name> --backup <file>`).
- [ ] Exportar salidas estructuradas adicionales (`status --json`).
- [ ] Añadir pruebas E2E automatizadas contra entorno k3s de staging.

---

## Notas

- Este roadmap reemplaza checklists históricos de fases previas ya cerradas.
- El estado fuente para prioridades futuras es este documento.

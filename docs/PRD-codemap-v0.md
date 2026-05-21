# PRD v0 — CodeMap AI Utility

## 1. Resumen
CodeMap es una utilidad local (Go + SQLite) que construye un mapa navegable del código de un proyecto para humanos y CLIs de IA (Pi, Claude Code, Codex). Su objetivo es entregar contexto estructurado, intención del código y trazabilidad histórica por símbolo.

## 2. Problema
Las IAs de desarrollo suelen trabajar con contexto parcial: entienden archivos aislados pero pierden continuidad de arquitectura, intención y evolución histórica.

## 3. Objetivo del producto
Permitir que cualquier agente o desarrollador consulte rápidamente:
- Qué hace un símbolo/módulo.
- Por qué existe (intención).
- Qué lo afecta y qué afecta.
- Cómo cambió en el tiempo (commits).

## 4. Usuarios objetivo
- Desarrolladores que trabajan en repos medianos/grandes.
- Agentes CLI (Pi/Claude/Codex) que necesitan contexto confiable y trazable.

## 5. Alcance MVP
Incluye:
- Indexado local de repositorio.
- Mapa tipo outline con navegación por símbolos.
- Relaciones de código (imports, calls, deps básicas).
- Historial de commits por archivo/símbolo.
- Notas de intención con evidencia y nivel de confianza.
- CLI con salida texto + JSON versionado.

No incluye (MVP):
- Sincronización cloud.
- Colaboración multiusuario en tiempo real.
- Agente IA secundario obligatorio.

## 6. Experiencia de uso (UX)
### 6.1 Vista principal
- Árbol lateral: proyecto → carpetas → archivos → símbolos.
- Panel detalle del nodo seleccionado:
  - Firma y documentación.
  - Callers / Callees.
  - Último commit + historial relevante.
  - Nota de intención (humana/IA), confianza, evidencia.

### 6.2 Búsqueda
- Búsqueda global por símbolo, archivo, texto de intención y commit.
- Resultado con salto directo al nodo.

### 6.3 Superficie de visualización
Fase MVP:
- TUI en terminal (principal).
- CLI JSON para consumo por herramientas externas.

Fase siguiente:
- Extensión VSCode como primera UI gráfica.

## 7. Interoperabilidad con CLIs de IA
Comandos base:
- `codemap index`
- `codemap query "<texto>"`
- `codemap symbol <id|name> --json`
- `codemap history <symbol>`
- `codemap impact <symbol>`

Contrato JSON:
- `schema_version` obligatorio.
- `evidence[]` obligatorio en respuestas explicativas.
- Campos mínimos: symbol, file, ranges, relations, intent, commits, confidence.

## 8. Arquitectura lógica
1. **Indexer**
   - Parsea código por lenguaje.
   - Extrae símbolos y relaciones.
   - Mantiene snapshots por estado/commit.

2. **Store (SQLite)**
   - Persistencia estructural + FTS5 para búsqueda textual.

3. **Intent Layer**
   - Registra intención por símbolo/módulo.
   - Fuentes: manual, commit message, PR text, IA.
   - Marca confianza y evidencia.

4. **Explorer**
   - TUI navegable.
   - Consultas CLI/JSON para integración con agentes.

## 9. Modelo de datos inicial
- `files(id, path, language, hash, snapshot_id)`
- `symbols(id, file_id, name, kind, signature, start_line, end_line)`
- `edges(id, from_symbol_id, to_symbol_id, edge_type)`
- `commits(hash, author, date, message)`
- `symbol_commits(id, symbol_id, commit_hash, change_type)`
- `intent_notes(id, target_type, target_id, source_type, note, confidence, evidence_ref, created_at)`
- `snapshots(id, repo_root, head_ref, created_at)`

## 10. Lenguajes objetivo
Fase 1 (MVP+):
- Go
- TypeScript/JavaScript
- Python
- Java
- C#
- Rust

Fase 2:
- PHP
- Ruby
- Kotlin

## 11. Reglas de trazabilidad
Toda respuesta que explique “qué/por qué/impacto” debe incluir evidencia:
- Archivo + rango de líneas.
- Commit hash relacionado.
- Extracto o referencia verificable.

Si una intención no tiene evidencia directa, debe mostrarse como hipótesis.

## 12. Actualización e incrementalidad
- Hash por archivo para detectar cambios.
- Reindexado parcial de archivos modificados.
- Snapshot nuevo al detectar cambio significativo o pedido explícito.

## 13. Privacidad y seguridad
- Local-first, sin red obligatoria.
- Sin exfiltración por defecto.
- Export explícito y opt-in para compartir contexto.

## 14. Criterio de éxito (demo)
El MVP se considera válido cuando:
1. Indexa un repo real sin configuración compleja.
2. Permite navegar símbolos y relaciones desde TUI.
3. Muestra historial de commits por símbolo.
4. Entrega intención + evidencia en panel y JSON.
5. Un CLI de IA externo consume `codemap symbol --json` y responde con mejor contexto.

## 15. Roadmap corto
- **R1**: Core indexer + SQLite schema + comandos base.
- **R2**: TUI navegable + intención/evidencia.
- **R3**: Historial por símbolo + impacto.
- **R4**: Extensión VSCode (visualización gráfica).

## 16. Riesgos principales
- Parse multilenguaje inconsistente.
- Mapeo símbolo↔commit con precisión variable.
- Ruido en intención inferida por IA.

Mitigación:
- Empezar con parsers confiables por lenguaje.
- Marcar confianza por fuente.
- Exigir evidencia visible para toda afirmación importante.

## 17. Reglas de ingeniería del proyecto
### 17.1 Control de versiones (Git)
- Todo cambio va por Git, sin trabajo fuera de versionado.
- Commits pequeños, atómicos y descriptivos.
- Convención sugerida de commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- Todo cambio funcional debe incluir actualización de docs técnicas si aplica.

### 17.2 Estrategia de ramas
- Rama principal protegida: `main`.
- Todo desarrollo en ramas de feature/bugfix:
  - `feat/<descripcion-corta>`
  - `fix/<descripcion-corta>`
  - `chore/<descripcion-corta>`
- Integración por Pull Request con revisión antes de merge.
- Evitar PRs gigantes; dividir cambios grandes en entregas revisables.

### 17.3 Calidad y escalabilidad de código
- Arquitectura modular por capas: `indexer`, `store`, `intent`, `explorer`, `cli`.
- Interfaces claras entre módulos para facilitar reemplazo/extensión.
- Evitar acoplamiento fuerte a un solo parser/lenguaje.
- Diseñar para incrementalidad (no full reindex por defecto).
- Mantener contratos JSON versionados y backward-compatible cuando sea posible.

### 17.4 Testing obligatorio
- Tests unitarios para parsing, persistencia SQLite y resolución de relaciones.
- Tests de integración para comandos CLI críticos (`index`, `symbol`, `history`, `impact`).
- Fixtures de repositorio para validar casos reales multilenguaje.
- Cada bug corregido debe agregar o actualizar un test que lo cubra.

### 17.5 Criterio de aceptación de cambios
Un cambio se acepta cuando:
1. Compila y pasa tests.
2. Mantiene trazabilidad (evidence) en respuestas.
3. No rompe contrato JSON vigente sin versionado explícito.
4. Incluye documentación mínima de uso si cambia comportamiento externo.

## 18. Definición de Done (DoD)
Para una feature del MVP:
- Código implementado y revisado por PR.
- Tests unitarios/integración agregados y en verde.
- Documentación de comando/flujo actualizada.
- Ejemplo reproducible ejecutado en un repo real.

## 19. Versionado y releases
- Estrategia SemVer: `MAJOR.MINOR.PATCH`.
- Regla de compatibilidad:
  - Cambios incompatibles en JSON/API de CLI requieren `MAJOR`.
  - Nuevas capacidades compatibles incrementan `MINOR`.
  - Fixes sin cambios de contrato incrementan `PATCH`.
- Mantener `CHANGELOG.md` por release con secciones `Added`, `Changed`, `Fixed`, `Deprecated`.
- Cada release debe incluir binarios para plataformas soportadas y notas de migración si aplica.

## 20. Manejo de errores y observabilidad
- Logs estructurados (JSON) con niveles: `debug`, `info`, `warn`, `error`.
- Salida CLI humana por defecto y modo máquina (`--json`) para automatización.
- Códigos de salida estables:
  - `0` éxito,
  - `1` error de ejecución,
  - `2` error de entrada/validación,
  - `3` error de datos/índice.
- En errores críticos, reportar contexto mínimo: comando, snapshot activo, archivo/símbolo implicado cuando exista.

## 21. Presupuestos de performance (objetivos iniciales)
Para repos de tamaño medio (~100k LOC):
- `codemap index` inicial: objetivo < 90s.
- Reindexado incremental tras cambio pequeño: objetivo < 10s.
- `codemap symbol` / `codemap history`: objetivo < 500ms.
- `codemap query` textual: objetivo < 1.5s.

Nota: son objetivos de diseño para guiar arquitectura; pueden ajustarse tras validación en repos reales.

## 22. Migraciones de SQLite
- Mantener `schema_migrations` con versión aplicada y fecha.
- Toda modificación de schema debe incluir script de migración `up` (y `down` cuando sea seguro).
- Cambios incompatibles en datos deben ir acompañados de plan de migración y fallback.
- El CLI debe detectar schema viejo y ofrecer migración explícita (`codemap migrate`).

## 23. Compatibilidad de plataformas
- Soporte objetivo: Linux y macOS en MVP.
- Soporte Windows en fase siguiente, con validación de paths/encoding.
- Evitar dependencias no portables o incluir fallback por OS.
- Tests de smoke por plataforma en CI al menos para comandos críticos.

## 24. Contribución y gobernanza
- Incluir `CONTRIBUTING.md` con flujo de ramas, estilo, tests y checklist de PR.
- Template de PR con:
  - objetivo del cambio,
  - impacto en contratos JSON,
  - evidencia de tests,
  - riesgos y rollback.
- Revisión obligatoria para cambios en schema, contratos CLI y parsers.
- Decisiones técnicas relevantes deben registrarse como ADRs livianas (`docs/adr/`).

## 25. Checklist de aceptación v1 (must-pass)
- `codemap index` crea índice SQLite sin configuración compleja.
- Reindexado incremental funcional (solo archivos cambiados).
- Mapa navegable muestra archivos y símbolos principales.
- `codemap symbol <x>` devuelve firma, ubicación y relaciones.
- `codemap impact <x>` devuelve callers/callees/imports relevantes.
- `codemap history <x>` muestra commits asociados al símbolo.
- Notas de intención soportadas con `source`, `confidence`, `evidence`.
- Respuestas explicativas incluyen `evidence[]`.
- Modo `--json` estable con `schema_version`.
- `codemap query` busca por símbolo/archivo/intención/commit.
- Códigos de error CLI estables y documentados.
- Tests unitarios e integración de comandos críticos en verde.

## 26. Requisitos de integración para agentes IA
### 26.1 Contrato de datos
- Salida JSON determinística, sin ruido en stdout cuando se usa `--json`.
- `schema_version` obligatorio en todas las respuestas de integración.
- Campos mínimos por consulta: `symbol`, `file`, `ranges`, `relations`, `commits`, `intent`, `confidence`, `evidence`.

### 26.2 Comandos requeridos
- `codemap index`
- `codemap symbol <id|name> --json`
- `codemap history <symbol> --json`
- `codemap impact <symbol> --json`
- `codemap query "<texto>" --json`

### 26.3 Trazabilidad y confianza
- Toda afirmación de impacto o intención debe incluir `evidence[]`.
- Intenciones sin evidencia directa se etiquetan como hipótesis.
- Confianza normalizada por fuente: `high` / `medium` / `low`.

### 26.4 Estado del índice
- El sistema debe exponer: `snapshot_id`, `head_ref`, `indexed_at`, `is_stale`.
- Si el índice está desactualizado, el comando debe advertirlo en la respuesta JSON.

### 26.5 Requisitos operativos para agentes
- Consultas objetivo en < 2s para flujo interactivo normal.
- Manejo de errores con códigos consistentes y mensaje parseable.
- Documentación breve de integración para prompts/herramientas externas.

## 27. Especificación JSON mínima (v1)
Todos los comandos en modo `--json` deben responder con esta envoltura base:
```json
{
  "schema_version": "1.0",
  "command": "symbol",
  "ok": true,
  "data": {},
  "errors": [],
  "meta": {
    "snapshot_id": "snap_123",
    "head_ref": "abc1234",
    "indexed_at": "2026-05-21T10:00:00Z",
    "is_stale": false
  }
}
```

Ejemplo mínimo `codemap symbol AuthService.login --json`:
```json
{
  "schema_version": "1.0",
  "command": "symbol",
  "ok": true,
  "data": {
    "symbol": {
      "id": "sym:ts:src/auth/service.ts:AuthService.login",
      "name": "AuthService.login",
      "kind": "method",
      "file": "src/auth/service.ts",
      "ranges": { "start_line": 42, "end_line": 88 }
    },
    "relations": {
      "callers": ["LoginController.handle"],
      "callees": ["TokenSigner.sign"]
    },
    "intent": {
      "note": "Centraliza validación y emisión de token",
      "source": "commit_msg",
      "confidence": "high"
    },
    "commits": ["a1b2c3d", "d4e5f6g"],
    "evidence": [
      { "type": "file_range", "file": "src/auth/service.ts", "start_line": 42, "end_line": 88 },
      { "type": "commit", "hash": "a1b2c3d" }
    ]
  },
  "errors": [],
  "meta": {
    "snapshot_id": "snap_123",
    "head_ref": "abc1234",
    "indexed_at": "2026-05-21T10:00:00Z",
    "is_stale": false
  }
}
```

## 28. Identidad canónica de símbolos
- Cada símbolo debe tener ID estable derivado de: `language + file logical path + qualified name + kind`.
- Para renombres/moves, conservar historial con tabla de alias:
  - `symbol_aliases(alias_id, symbol_id, previous_qualified_name, previous_path, valid_until_snapshot)`.
- Consultas por nombre deben resolver primero ID canónico, luego alias.

## 29. Precisión de mapeo commit↔símbolo
- Clasificar vínculo con `link_strength`: `strong`, `medium`, `weak`.
- `strong`: diff toca rango del símbolo en el commit.
- `medium`: diff toca el archivo y hay proximidad textual al símbolo.
- `weak`: inferencia contextual sin toque directo de rango.
- UI/JSON debe mostrar `link_strength`; por defecto priorizar solo `strong|medium`.

## 30. Fallback cuando parser falla
- Si un archivo no parsea, no abortar todo el indexado.
- Registrar incidente en tabla `parse_errors(file, parser, error, snapshot_id)`.
- Degradar a extracción superficial (nombre de archivo + texto indexable) cuando sea posible.
- Reportar resumen de archivos fallidos al final de `codemap index`.

## 31. Seguridad local y límites de lectura
- Por defecto excluir rutas: `.git/`, `node_modules/`, `vendor/`, `dist/`, `build/`, binarios y archivos > umbral configurable.
- Soportar `.codemapignore` para exclusiones custom.
- Nunca leer ni exponer secretos por inferencia activa; solo rutas explícitamente indexadas.
- Modo estricto opcional: indexar solo extensiones permitidas por configuración.

## 32. Bootstrap de intención
- Inicializar intención desde:
  1. mensajes de commit,
  2. README/docs arquitectónicas,
  3. notas manuales del usuario (`codemap note add`).
- Las notas IA deben quedar marcadas `source=ai_generated` y `confidence` no mayor a `medium` sin evidencia adicional.
- Permitir revisión humana para promover confianza (`promote confidence`) con auditoría.

## 33. Matriz de riesgos, mitigación y aceptación
1. **Riesgo**: contrato JSON ambiguo.
   - **Mitigación**: spec fija + ejemplos por comando.
   - **Aceptación**: clientes Pi/Claude/Codex consumen sin parsers ad-hoc.

2. **Riesgo**: pérdida de trazabilidad por renombres.
   - **Mitigación**: ID canónico + alias por snapshot.
   - **Aceptación**: `history` mantiene continuidad tras rename/move.

3. **Riesgo**: asociaciones commit↔símbolo engañosas.
   - **Mitigación**: `link_strength` visible y filtrable.
   - **Aceptación**: UI y JSON distinguen fuerte/medio/débil.

4. **Riesgo**: indexado frágil por parse errors.
   - **Mitigación**: fallback parcial + registro estructurado de errores.
   - **Aceptación**: indexado finaliza aunque fallen archivos aislados.

5. **Riesgo**: exposición accidental de contenido sensible.
   - **Mitigación**: exclusiones por defecto + `.codemapignore` + modo estricto.
   - **Aceptación**: rutas sensibles no aparecen en índice por defecto.

6. **Riesgo**: capa de intención vacía o poco útil al inicio.
   - **Mitigación**: bootstrap desde commits/docs + notas manuales.
   - **Aceptación**: símbolos críticos muestran al menos una intención inicial trazable.

## 34. CodeMap Enforcement Protocol (uso obligatorio por agentes IA)
Objetivo: asegurar que cualquier IA (Pi/Claude/Codex u otra) use CodeMap de forma consistente y no opcional.

### 34.1 Pre-hook obligatorio (antes de responder sobre código)
Antes de responder consultas técnicas, el runtime debe ejecutar al menos una lectura de CodeMap:
1. `codemap symbol` cuando hay símbolo objetivo.
2. `codemap impact` cuando se consulta impacto.
3. `codemap history` cuando se consulta evolución.
4. `codemap query` cuando la pregunta es abierta.

Regla: no emitir respuesta final sobre código sin evidencia obtenida desde CodeMap en el turno actual.

### 34.2 Post-hook obligatorio (después de trabajo relevante)
Después de cambios importantes, el runtime debe:
1. refrescar índice (`codemap index` incremental),
2. actualizar intención/nota si aplica (`codemap note add/update`),
3. dejar trazabilidad de evidencia para próximos turnos.

### 34.3 Policy gate (bloqueo por incumplimiento)
El orquestador debe bloquear la respuesta final (`deny`) cuando:
- no hubo lectura de CodeMap,
- no hay `evidence[]` en respuestas explicativas,
- el índice está stale y no se aceptó continuar así.

### 34.4 Política de falla (fail-closed)
Si CodeMap está en modo requerido y no está disponible:
1. detener respuesta técnica final,
2. reportar “CodeMap no disponible”,
3. pedir confirmación explícita para continuar en modo degradado.

No se debe inventar contexto como sustituto de CodeMap.

### 34.5 Señales de cumplimiento (metadata)
Cada ejecución crítica debe reportar:
- `codemap_read_performed: true|false`
- `codemap_write_performed: true|false`
- `codemap_evidence_attached: true|false`
- `codemap_mode: required|best_effort|disabled`

### 34.6 Modos de operación
- `required`: uso de CodeMap obligatorio con bloqueo por incumplimiento.
- `best_effort`: intenta usar CodeMap; si falla, continúa con advertencia explícita.
- `disabled`: sin CodeMap (solo por decisión explícita del usuario).

Modo recomendado para este proyecto: `required` en desarrollo normal, `best_effort` solo en contingencia.

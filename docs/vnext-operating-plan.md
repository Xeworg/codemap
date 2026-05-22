# CodeMap vNext — Plan Operativo Completo

Base: `docs/prd-vnext.md`  
Contexto de ejecución: equipo chico (vos + el Gentleman), baja burocracia, foco en shipping iterativo con calidad.

## 1) Objetivo de ejecución
Entregar vNext con tres capacidades productivas y utilizables en flujo real:
1. `codemap impact` confiable para planear cambios
2. Explain-why-not-found en `symbol/history`
3. `codemap deadcode` v1 (reporte + sugerencias)

## 2) Principios de trabajo
- Una sola fuente de verdad funcional: PRD + este plan.
- Iteraciones cortas con evidencia (tests + smoke).
- Sin issues formales por ahora; usar checklist por milestone.
- Cambios chicos y revisables (evitar diffs gigantes).
- Cada feature nueva actualiza docs y skill en el mismo ciclo.

## 3) Secuencia recomendada (orden real)
1. **Fase A — Contratos y taxonomías (M1)**
2. **Fase B — Explain-not-found (M3)**
3. **Fase C — Impact v1 (M2)**
4. **Fase D — Deadcode v1 (M4)**
5. **Fase E — Hardening, docs, skill y release (M5)**

---

## 4) Plan por fase

### Fase A — M1 Contratos y taxonomías
**Resultado esperado**: schemas y vocabulario estables antes de codificar features.

**Entregables**
- Contrato JSON para `impact`
- Contrato JSON para `deadcode`
- Taxonomía `not_found_cause` para `symbol/history`
- Definición de `risk_tier`, `confidence`, `suggestion`

**Checklist**
- [ ] Actualizar `docs/codemap-cli-json-contract.md`
- [ ] Definir ejemplos válidos + errores
- [ ] Alinear envelope (`ok/data/errors/meta`)
- [ ] Nota de compatibilidad para consumidores actuales

**DoD (Definition of Done)**
- [ ] Contratos claros y sin ambigüedad en resultados vacíos
- [ ] Todos los nuevos enums documentados

---

### Fase B — M3 Explain-not-found
**Resultado esperado**: cuando `symbol/history` no encuentra algo, devuelve causa y acción.

**Entregables**
- Explain path en `symbol` y `history`
- `causes[]` + `recommended_actions[]`

**Checklist**
- [ ] Implementar detección de causas típicas
- [ ] Exponer códigos máquina + mensaje humano
- [ ] Tests de integración para casos de no hallazgo
- [ ] Ejemplos de uso en help/docs

**DoD**
- [ ] Casos de no hallazgo con salida accionable
- [ ] Tests cubren al menos 4 causas distintas

---

### Fase C — M2 Impact v1
**Resultado esperado**: comando útil para pre-PR y evaluación de riesgo.

**Entregables**
- `codemap impact <symbol>`
- Lista de impacto por `high|medium|low`
- Evidencia por item y orden determinístico

**Checklist**
- [ ] Wiring de comando en CLI
- [ ] Query de dependencias/referencias
- [ ] Heurística de risk tier
- [ ] Cap de resultados por defecto
- [ ] Unit tests + integración + smoke local

**DoD**
- [ ] JSON válido y estable
- [ ] Resultado entendible para flujo de PR

---

### Fase D — M4 Deadcode v1 (reporte + sugerencias)
**Resultado esperado**: detectar probable código muerto sin fricción de enforcement.

**Entregables**
- `codemap deadcode`
- Clasificación: `unused|likely-unused|uncertain`
- Sugerencia: `remove|deprecate|justify`

**Checklist**
- [ ] Detección base por cero referencias entrantes
- [ ] Reglas de clasificación/confianza
- [ ] Exclusion list (generated, entrypoints, allowlist)
- [ ] Fixtures de validación y medición de falsos positivos

**DoD**
- [ ] Salida con símbolo, ubicación, clase, sugerencia, evidencia, confianza
- [ ] Falsos positivos medidos/documentados en muestra curada

---

### Fase E — M5 Hardening + Docs + Skill + Release
**Resultado esperado**: adopción simple y release usable.

**Entregables**
- Docs funcionales actualizadas
- Skill actualizada para nuevos workflows
- Smoke checklist completo
- Release notes

**Checklist**
- [ ] Actualizar `docs/codemap-cli-json-contract.md`
- [ ] Actualizar `integrations/pi/skills/codemap-usage/SKILL.md`
- [ ] Verificar `codemap install` copia skill actualizada
- [ ] Correr smoke: `index/symbol/history/impact/deadcode`
- [ ] Publicar notas de release con límites conocidos

**DoD**
- [ ] Skill y docs reflejan comportamiento real
- [ ] Smoke en entorno limpio pasa

---

## 5) Cadencia sugerida
- **Bloque 1 (día 1-2):** Fase A + B
- **Bloque 2 (día 3-4):** Fase C
- **Bloque 3 (día 5-6):** Fase D
- **Bloque 4 (día 7):** Fase E + estabilización

> Ajustable según complejidad real; priorizar terminar verticalmente cada fase.

## 6) Estrategia de calidad
- Test-first cuando sea posible para lógica nueva.
- Cada fase termina con:
  1) tests relevantes,
  2) salida JSON de ejemplo,
  3) mini smoke manual.
- Si aparece ruido alto en resultados, frenar y ajustar heurísticas antes de seguir.

## 7) Riesgos de ejecución y respuesta
- **Ruido/falsos positivos** (impact/deadcode): usar `uncertain`, exclusiones, confidence.
- **Deriva de contratos**: mantener M1 como gate previo a features.
- **Sobrecarga de cambio**: evitar mezclar 2 features grandes en un mismo commit.

## 8) Convención de commits (simple)
- 1 commit por unidad lógica revisable.
- Mensaje sugerido:
  - `feat(codemap): add explain-not-found causes for symbol/history`
  - `feat(codemap): add impact command with risk tiers`
  - `feat(codemap): add deadcode report with suggestions`
  - `docs(codemap): update skill and CLI JSON contract for vNext`

## 9) Criterio de “listo para arrancar código”
Podemos arrancar implementación cuando:
- [ ] Aceptaste este plan
- [ ] Fase A queda cerrada (contratos)
- [ ] Decidimos primer slice técnico (recomendado: Explain-not-found)

## 10) Próximo paso inmediato recomendado
Empezar por **Fase A (M1)** y cerrar contrato de `not_found_cause` + ejemplos JSON hoy mismo. Luego entrar directo a Fase B.

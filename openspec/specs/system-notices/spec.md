# system-notices Specification

## Purpose

Orquestar notificaciones de sistema transversales (prioridad, scope, una visible a la vez) como paquete workspace reutilizable, desacoplado del dominio de cada app consumidora.

## Requirements

### Requirement: Paquete workspace independiente

El orquestador de system notices MUST publicarse como `@bowerbird/system-notices` en `packages/system-notices/`. MUST ser consumible vía `workspace:*` desde apps del monorepo. MUST NOT depender de código de aplicaciones consumidoras.

#### Scenario: Consumo desde PWA

- **WHEN** `apps/pwa` declara dependencia `@bowerbird/system-notices`
- **THEN** puede importar `SystemNotice`, `SystemNoticesOrchestrator` y el host Angular sin acoplar el package a `pwa-install`

### Requirement: Contrato SystemNotice

El paquete MUST exportar la interfaz `SystemNotice` con `id`, `priority`, `scope` (`global` | `tenant`), `canShow()`, `show()` y `dismiss(reason)`. Las implementaciones concretas MUST registrarse en el composition root del consumidor.

#### Scenario: Notice registrada por consumer

- **WHEN** la PWA registra una notice de instalación vía `provideSystemNotices()`
- **THEN** el orchestrator la evalúa según prioridad y scope sin conocer su lógica interna

### Requirement: Una notice visible

El orchestrator MUST garantizar que solo una notice se muestra a la vez. Al resolver o dismissar la activa, MUST evaluar la siguiente elegible por prioridad descendente.

#### Scenario: Cola por prioridad

- **WHEN** dos notices con distinta prioridad son elegibles simultáneamente
- **THEN** solo se muestra la de mayor prioridad hasta que se resuelva

### Requirement: Host Angular con filtro por scope

El paquete MUST exportar un componente host (`@bowerbird/system-notices/angular`) que acepte `scope: 'global' | 'tenant'` y evalúe únicamente notices de ese scope.

#### Scenario: Host tenant

- **WHEN** el host se monta con `scope="tenant"`
- **THEN** solo evalúa notices cuyo `scope` es `tenant`

### Requirement: Orchestrator testeable sin DOM

El núcleo del orchestrator (`SystemNoticesOrchestrator`) MUST ser framework-agnostic y testeable con notices fake sin TestBed ni browser APIs.

#### Scenario: Test unitario aislado

- **WHEN** se ejecutan tests del package con notices mock
- **THEN** pasan sin levantar `apps/pwa` ni Angular TestBed para el orchestrator core

### Requirement: Aislamiento de estado del orchestrator

El package MUST ser el único dueño de la cola de notices activa en memoria (sesión). MUST NOT leer ni escribir storage del browser (`localStorage`, `sessionStorage`) ni estado de dominio del consumidor.

#### Scenario: Sin reach-through a storage del consumer

- **WHEN** el orchestrator evalúa qué notice mostrar
- **THEN** delega elegibilidad a `canShow()` de cada notice concreta sin acceder a claves de persistencia del consumer

### Requirement: Superficie pública mínima

El entry `./` del package MUST exportar únicamente `SystemNotice`, `SYSTEM_NOTICE` (token), `SystemNoticesOrchestrator` y tipos de scope/priority. MUST NOT re-exportar implementación interna de la cola ni helpers privados.

#### Scenario: Consumer importa API pública

- **WHEN** `pwa-install` implementa una notice
- **THEN** solo importa desde `@bowerbird/system-notices` (port + token), no desde paths internos del package

### Requirement: Fail independence en ciclo de notices

Un error en `show()` o `dismiss()` de una notice concreta MUST NOT impedir que el orchestrator evalúe o muestre otras notices elegibles.

#### Scenario: Notice con show() fallido

- **WHEN** `show()` de una notice lanza un error
- **THEN** el orchestrator lo contiene y puede presentar la siguiente notice elegible por prioridad

### Requirement: Observabilidad atribuible

El orchestrator MUST emitir eventos diagnósticos con prefijo `system_notice_` (p. ej. `system_notice_shown`, `system_notice_dismissed`) atribuibles al package, sin mezclar eventos de dominio PWA (`pwa_*`).

#### Scenario: Notice presentada

- **WHEN** el orchestrator activa una notice
- **THEN** se emite `system_notice_shown` con `noticeId` y `scope`

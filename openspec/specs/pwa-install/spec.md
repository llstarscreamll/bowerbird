# pwa-install Specification

## Purpose

Promover la instalación de Bowerbird como PWA de forma contextual y no invasiva, respetando el engagement del usuario y ofreciendo acceso permanente desde el menú del tenant.

## Requirements

### Requirement: Invariantes solo en el aggregate

Las reglas de elegibilidad (2ª visita en 7 días), cooldowns y silenciamiento MUST encapsularse en `InstallEngagement` y sus VOs. Los adapters `SystemNotice` (`pwa-install-chromium`, `pwa-install-ios`, `pwa-update`) MUST delegar en coordinator/commands y MUST NOT implementar reglas de engagement directamente.

#### Scenario: Notice delega elegibilidad

- **WHEN** el orchestrator invoca `canShow()` en la notice de instalación Chromium
- **THEN** la notice consulta `PwaInstallCoordinator.canShowAutoPrompt()` (que a su vez consulta el aggregate) y no lee `localStorage` ni evalúa `visits.length` por su cuenta

### Requirement: Reconstitución desde persistencia

El aggregate `InstallEngagement` MUST reconstituirse desde storage vía factory `reconstitute()` sin setters públicos. El repository MUST serializar/deserializar el aggregate completo, no DTOs mutables sueltos.

#### Scenario: Carga tras reload

- **WHEN** el repository carga engagement desde `bb:pwa:*`
- **THEN** devuelve un `InstallEngagement` reconstituido listo para intent methods sin estado parcial expuesto

### Requirement: Sin banner persistente de instalación

La PWA MUST NOT mostrar un card o banner fijo de instalación que permanezca visible sin acción del usuario ni opción de cerrar.

#### Scenario: Usuario navega sin instalar

- **WHEN** el usuario está en una ruta de tenant y no ha instalado la app ni interactuado con la promoción
- **THEN** no hay ningún elemento de UI de instalación fijo en pantalla de forma permanente

### Requirement: Promoción solo dentro de tenant

La promoción automática de instalación MUST mostrarse únicamente cuando el usuario está autenticado y navega dentro del tenant layout (ruta `/:tenantId/*`). MUST NOT mostrarse en login, lobby, onboarding ni rutas globales sin `tenantId`.

#### Scenario: Usuario en lobby

- **WHEN** un usuario autenticado visita `/lobby`
- **THEN** no se muestra promoción automática de instalación

#### Scenario: Usuario en tenant

- **WHEN** un usuario autenticado navega a `/:tenantId/dashboard` y cumple los demás criterios de elegibilidad
- **THEN** el sistema puede mostrar la promoción de instalación según las reglas de engagement y plataforma

### Requirement: Gate de engagement — 2ª visita en 7 días

El sistema MUST registrar visitas por sesión de navegador y MUST considerar elegible para promoción automática solo cuando el usuario tiene al menos 2 visitas registradas y la segunda ocurre dentro de los 7 días calendario posteriores a la primera visita registrada.

#### Scenario: Primera visita

- **WHEN** el usuario abre la PWA por primera vez en ese navegador
- **THEN** se registra la visita y no se muestra promoción automática de instalación

#### Scenario: Segunda visita dentro de ventana

- **WHEN** el usuario inicia una nueva sesión y ya tiene una visita previa registrada hace 3 días
- **THEN** el usuario es elegible para promoción automática (si cumple demás criterios)

#### Scenario: Segunda visita fuera de ventana

- **WHEN** el usuario inicia una nueva sesión y su primera visita fue hace más de 7 días
- **THEN** el usuario no es elegible para promoción automática (solo pull model en menú)

### Requirement: Presentación responsive de la promoción

En plataformas Chromium con `beforeinstallprompt` capturado, el sistema MUST mostrar un snackbar auto-dismiss (≈6 s) en viewport desktop y un bottom sheet en viewport mobile. El copy MUST ser conciso:

- **Título:** «Instala Bowerbird»
- **Cuerpo:** «Tu espacio de trabajo, a un toque.»
- **Acción primaria:** «Instalar»
- **Acción secundaria (mobile sheet):** «Continuar en navegador»
- **Dismiss:** «Ahora no»

#### Scenario: Desktop elegible

- **WHEN** un usuario elegible usa viewport desktop dentro de tenant
- **THEN** se muestra snackbar con título, cuerpo y acciones «Instalar» / «Ahora no», y auto-dismiss tras ~6 s

#### Scenario: Mobile elegible

- **WHEN** un usuario elegible usa viewport mobile dentro de tenant
- **THEN** se muestra bottom sheet con título, cuerpo y acciones «Instalar», «Continuar en navegador» y «Ahora no»

### Requirement: Guía de instalación iOS

En iOS Safari (sin `beforeinstallprompt`), cuando el usuario es elegible y la app no está en modo standalone, el sistema MUST mostrar un bottom sheet instructivo en lugar del prompt nativo. Copy:

- **Título:** «Añade Bowerbird a tu inicio»
- **Cuerpo:** pasos numerados: Compartir → «Añadir a pantalla de inicio» → Confirmar
- **Cierre:** «Entendido»

#### Scenario: iOS elegible

- **WHEN** un usuario elegible en iOS Safari navega dentro de tenant y la app no está instalada
- **THEN** se muestra el sheet instructivo con el copy definido

### Requirement: Cooldowns tras dismiss

El sistema MUST persistir preferencias de dismiss y MUST respetar cooldowns progresivos:

| Acción                                 | Cooldown                                   |
| -------------------------------------- | ------------------------------------------ |
| Timeout del snackbar (sin interacción) | 3 días                                     |
| «Ahora no» o «Continuar en navegador»  | 7 días                                     |
| Segundo dismiss explícito acumulado    | 30 días                                    |
| Tercer dismiss explícito               | Solo pull model (sin promoción automática) |

#### Scenario: Usuario elige «Ahora no»

- **WHEN** el usuario dismissa la promoción con «Ahora no»
- **THEN** no se muestra promoción automática durante 7 días y se incrementa el contador de dismiss

#### Scenario: Tercer dismiss

- **WHEN** el usuario ha dismissado explícitamente 3 veces
- **THEN** no se muestra más promoción automática; solo queda disponible el ítem del menú

### Requirement: Decline atómico de auto-prompt

Un dismiss de la promoción automática MUST aplicar en una sola operación el cooldown correspondiente y el incremento del contador de dismiss. MUST NOT quedar estado parcial (p. ej. cooldown activo sin incrementar contador, o viceversa).

#### Scenario: Usuario elige «Continuar en navegador»

- **WHEN** el usuario dismissa la promoción mobile con «Continuar en navegador»
- **THEN** se registra un dismiss explícito, se activa cooldown de 7 días y el contador de dismiss refleja exactamente un incremento respecto al estado previo

### Requirement: Domain events de engagement

Tras mutaciones de engagement (`recordSessionVisit`, `declineAutoPrompt`), el sistema MUST emitir domain events internos que la capa application traduce a eventos de analytics. Los eventos MUST incluir al menos: `SessionVisitRecorded`, `AutoPromptDeclined`, y `AutoPromptBecameEligible` cuando el umbral de visitas se cumple por primera vez.

#### Scenario: Segunda visita genera elegibilidad

- **WHEN** `recordSessionVisit` hace que el usuario cumpla el umbral de 2ª visita en 7 días
- **THEN** se emite `AutoPromptBecameEligible` y, vía application handler, el evento de analytics `pwa_install_eligible`

#### Scenario: Decline genera evento

- **WHEN** el usuario dismissa con «Ahora no»
- **THEN** se emite `AutoPromptDeclined` y, vía application handler, `pwa_install_prompt_action` con `action: not_now`

### Requirement: Pull model en menú de usuario

Mientras la app sea instalable (Chromium) o aplicable la guía iOS, el menú de usuario del tenant layout MUST incluir un ítem «Instalar aplicación» que dispare el prompt nativo o la guía iOS sin requerir engagement gate ni cooldown.

#### Scenario: Acceso desde menú

- **WHEN** el usuario abre el menú de usuario dentro de tenant y la app es instalable
- **THEN** ve el ítem «Instalar aplicación» y puede iniciar instalación o guía al seleccionarlo

### Requirement: System notices unificado

Las notificaciones de sistema (actualización de service worker e instalación PWA) MUST gestionarse mediante el paquete workspace `@bowerbird/system-notices`, que expone el contrato `SystemNotice` y un orquestador que muestra una sola notice a la vez por prioridad. Cada notice MUST declarar un `scope` (`global` o `tenant`). Las notices de instalación MUST tener scope `tenant`; la de actualización MUST tener scope `global`. La notice de actualización MUST tener prioridad sobre la de instalación. El paquete MUST NOT depender de código de dominio Bowerbird ni de `pwa-install`.

#### Scenario: Update e install elegibles simultáneamente

- **WHEN** hay una actualización disponible y el usuario es elegible para promoción de instalación
- **THEN** solo se muestra la notice de actualización hasta que se resuelva o dismiss

#### Scenario: Scope tenant no muestra notices globales en lobby

- **WHEN** el host de notices está montado con scope `tenant` y el usuario está en `/lobby`
- **THEN** no se evalúan ni muestran notices con scope `tenant` (incluida instalación)

#### Scenario: Package aislado del dominio PWA

- **WHEN** se inspeccionan las dependencias de `@bowerbird/system-notices`
- **THEN** no hay imports desde `apps/pwa` ni módulos de negocio Bowerbird

### Requirement: Registro de notices en composition root

Las notices concretas (`pwa-update`, `pwa-install-chromium`, `pwa-ios-install`) MUST registrarse en `app.config.ts` vía `providePwaInstall()` usando el token `SYSTEM_NOTICE` (`multi: true`). MUST NOT instanciarse el orchestrator desde `pwa-install`.

#### Scenario: Wiring centralizado

- **WHEN** se revisa el bootstrap de la PWA
- **THEN** `provideSystemNotices()` y `providePwaInstall()` se declaran en `app.config.ts` y ningún layout registra notices

### Requirement: Aislamiento de estado de engagement

Las claves de persistencia de engagement (`bb:pwa:visits`, `bb:pwa:install-prefs`, `bb:pwa:session`) MUST ser escritas únicamente por el adapter de storage de `pwa-install`. Ningún layout, componente de presentación ni otro módulo MUST acceder a esas claves directamente.

#### Scenario: Layout delega en coordinator

- **WHEN** el tenant layout necesita saber si mostrar el ítem de menú «Instalar aplicación»
- **THEN** consulta el coordinator público de `pwa-install` y no lee `localStorage` directamente

### Requirement: Analytics no bloqueante

El envío de eventos de funnel MUST NOT interrumpir ni retrasar la promoción de instalación, el prompt nativo ni la navegación del usuario. Un fallo en el adapter de analytics MUST ser contenido (sin propagar error al usuario).

#### Scenario: Adapter de analytics falla

- **WHEN** el adapter de analytics lanza un error interno al emitir un evento
- **THEN** la acción de instalación o dismiss del usuario se completa con normalidad

### Requirement: No promover si ya instalada

El sistema MUST NOT mostrar promoción automática ni guía iOS cuando la app corre en `display-mode: standalone` (ya instalada).

#### Scenario: App instalada

- **WHEN** el usuario abre Bowerbird en modo standalone
- **THEN** no se muestra promoción de instalación ni guía iOS

### Requirement: Funnel de analytics (fase 1 — cliente)

El sistema MUST emitir eventos de analytics del funnel de instalación. En fase 1 MUST registrarlos vía abstracción cliente (`console.debug` en desarrollo). Eventos mínimos:

- `pwa_visit_recorded`
- `pwa_install_eligible`
- `pwa_install_prompt_shown` (variant: `snackbar` | `sheet` | `ios_guide`)
- `pwa_install_prompt_action` (action: `install` | `not_now` | `continue_browser` | `timeout`)
- `pwa_install_native_result` (outcome: `accepted` | `dismissed`)
- `pwa_installed`
- `pwa_install_menu_clicked`
- `pwa_update_prompt_shown`
- `pwa_update_prompt_accepted`

#### Scenario: Promoción mostrada

- **WHEN** se muestra una promoción de instalación al usuario
- **THEN** se emite `pwa_install_prompt_shown` con la variante correspondiente

#### Scenario: Usuario instala

- **WHEN** el browser dispara `appinstalled`
- **THEN** se emite `pwa_installed`

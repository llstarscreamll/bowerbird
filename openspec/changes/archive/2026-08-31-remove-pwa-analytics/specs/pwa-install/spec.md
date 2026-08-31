## REMOVED Requirements

### Requirement: Domain events de engagement

**Reason**: Los domain events (`SessionVisitRecorded`, `AutoPromptBecameEligible`, `AutoPromptDeclined`) existían únicamente para alimentar el funnel de analytics. Sin analytics propio, no aportan valor y añaden complejidad al aggregate y a los commands.

**Migration**: Las mutaciones de engagement (`recordSessionVisit`, `declineAutoPrompt`) persisten el aggregate sin emitir eventos. La elegibilidad y los cooldowns siguen calculándose en el aggregate y sus VOs.

### Requirement: Analytics no bloqueante

**Reason**: No hay adapter ni port de analytics que proteger. El requirement era específico de la infraestructura `AnalyticsPort` eliminada.

**Migration**: Ninguna acción de usuario dependía de analytics para completarse; el comportamiento observable no cambia.

### Requirement: Funnel de analytics (fase 1 — cliente)

**Reason**: El funnel de instalación PWA (`pwa_visit_recorded`, `pwa_install_prompt_shown`, `pwa_installed`, etc.) se descarta en favor de integración futura con vendor externo (PostHog u otro) cuando haya usuarios reales y múltiples flujos que medir.

**Migration**: No se emiten eventos de funnel desde la PWA. La promoción, engagement gate, cooldowns y pull model permanecen sin cambios. Cuando se integre un vendor, instrumentar puntos clave con `capture()` directo sin reintroducir `AnalyticsPort` prematuramente.

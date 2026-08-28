## Context

El frontend (Angular PWA) actualmente utiliza el término "Contrapartes" para referirse a la entidad de negocio `Party`. Se requiere un cambio cosmético a "Contactos" sin afectar el diseño del backend ni la base de datos (ver `proposal.md`).

## Goals / Non-Goals

**Goals:**

- Actualizar todas las instancias estáticas de "Contraparte" a "Contacto" en los archivos TypeScript y HTML del frontend.
- Mantener la integridad de los componentes visuales existentes.

**Non-Goals:**

- No se creará un sistema de internacionalización (i18n). Los textos seguirán estáticos.
- No se renombrarán las rutas en el frontend (seguirá siendo `/parties`), ni los estados o clases internas (seguirá siendo `parties.store.ts`).

## Decisions

**1. Alcance de los cambios textuales (UI Only)**

- **Decisión:** Los reemplazos de texto se limitarán a etiquetas visibles para el usuario en la interfaz.
- **Alternativas consideradas:** Renombrar también las variables internas, rutas del PWA y directorios para hacer coincidir `parties` con `contacts`. _Rechazado_ porque rompe la alineación con el backend y el concepto central del negocio (Party/Tercero) sin justificación.

**2. Actualización de Glosario**

- **Decisión:** Modificar explícitamente el `GLOSSARY.md` para asentar que "Contacto" es la etiqueta UI para un `Party`, el cual contablemente representa un "Tercero".

## Risks / Trade-offs

- **Risk:** Olvidar instancias ocultas en modales o tooltips. → _Mitigación:_ Se utilizará búsqueda exhaustiva (grep) por "contraparte" en todo `apps/pwa` para asegurar cobertura completa.

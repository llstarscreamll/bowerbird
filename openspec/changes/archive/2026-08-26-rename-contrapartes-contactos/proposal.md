## Why

El término "Contrapartes" resulta confuso y demasiado formal/legal para los usuarios finales de un sistema de facturación y CRM. Cambiarlo a "Contactos" mejora significativamente la experiencia de usuario en el frontend, resultando mucho más intuitivo. El backend unificado y el dominio mantendrán el modelo de `Party` que soporta la lógica contable y de roles sin duplicar datos.

## What Changes

- Renombrar el término "Contrapartes" a "Contactos" en la UI del frontend (PWA). Esto incluye la navegación lateral en el `tenant-layout`, vistas principales (`master.page.ts`), botones, columnas de tablas y mensajes emergentes (toasts) del estado `parties.store.ts`.
- Renombrar las referencias visuales de "Contraparte" a "Contacto" en el detalle de la factura (`detail.page.ts`).
- Actualizar `docs/domain/GLOSSARY.md` para eliminar "Contraparte" y establecer "Contacto / Tercero" como el término de negocio, asignado a la entidad `Party`.
- **No goals / Out of scope:** No se realizarán cambios en la base de datos, arquitectura de backend, ni en las rutas de la API, que seguirán utilizando el módulo `parties`.

## Capabilities

### New Capabilities

_(Ninguna)_

### Modified Capabilities

- `parties`: Actualización de terminología en la UI (de "Contrapartes" a "Contactos") y glosario de negocio.

## Impact

- Frontend: Componentes de layout, menú principal y módulo de Parties y Facturas (`apps/pwa`).
- Documentación: Glosario de dominio (`docs/domain/GLOSSARY.md`).
- Backend: Sin impacto.

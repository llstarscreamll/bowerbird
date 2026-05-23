# Documentación de producto (negocio)

Esta capa documenta el _que_ y el _por que_ de las funcionalidades para negocio y usuarios.

## Estado actual

Actualmente no hay funcionalidades de negocio formalmente definidas. Esta carpeta deja la estructura lista para crecer con una convención clara.

## Convenciones

- Un archivo por funcionalidad de negocio en `docs/product/features/`.
- Nombres sugeridos: `YYYY-MM-slug-funcionalidad.md`.
- Toda funcionalidad debe incluir alcance, reglas, actores, eventos, métricas y criterios de aceptación.
- Referenciar implementación técnica desde aqui, no duplicar detalles de código.

## Estructura recomendada

- `features.md`: catálogo resumido de funcionalidades y estado.
- `features/`: especificaciones por funcionalidad.
- `_templates/`: plantillas para nuevas funcionalidades.

## Flujo recomendado para nuevas funcionalidades

1. Crear spec de negocio desde plantilla `docs/product/_templates/feature-spec.md`.
2. Acordar criterios de aceptación y métricas.
3. Enlazar impacto técnico en `docs/technical/`.
4. Mantener `features.md` como índice vivo.

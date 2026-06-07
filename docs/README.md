# Documentación del proyecto

Este directorio separa la documentación en dos grandes dominios para escalar sin mezclar audiencias.

## 1) Documentación técnica (`docs/technical/`)

Para implementación, operación, arquitectura y calidad de código.

- [Getting started técnico](./technical/getting-started.md)
- [Tooling: CodeGraph](./technical/tooling/codegraph.md)
- [Tooling: LocalStack](./technical/tooling/localstack.md)
- [Arquitectura backend](./technical/architecture/backend-api.md)
- [Arquitectura: eventos vs jobs](./technical/architecture/events-vs-jobs.md)
- [Arquitectura frontend](./technical/architecture/frontend-web.md)
- [Despliegue AWS](./technical/deployment/aws.md)
- [Calidad de desarrollo](./technical/quality/development-quality.md)

## 2) Documentación de producto/negocio (`docs/product/`)

Para funcionalidades de negocio, reglas, flujos de usuario, roadmap y decisiones de producto.

- [Convenciones de documentación de producto](./product/README.md)
- [Catálogo de funcionalidades de negocio](./product/features.md)

## Convención recomendada

- Técnico: todo lo que responde _cómo funciona y cómo operarlo_.
- Producto: todo lo que responde _qué valor entrega, para quién y por qué_.
- Evitar duplicar contenido entre ambas capas; enlazar en lugar de copiar.

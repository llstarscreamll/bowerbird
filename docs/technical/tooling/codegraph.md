# Tooling: CodeGraph

## Propósito

Este proyecto usa CodeGraph para indexar el código fuente y permitir exploración estructural rápida del repo (símbolos, callers/callees, impacto de cambios y contexto de arquitectura) sin depender de búsquedas textuales manuales.

## Recomendación de uso

- Usarlo para entender flujo de llamadas, dependencias e impacto antes de tocar código.
- Preferir CodeGraph para preguntas estructurales; usar grep/read para textos literales.

## Cuando reindexar

Ejecuta nuevamente la indexación cuando:

- El repositorio no tenga `.codegraph/`.
- Cambies a una rama con muchos cambios estructurales.
- Hagas refactors grandes (muchos archivos/símbolos renombrados).
- Notes resultados desactualizados o errores de símbolos no encontrados.

Comando recomendado:

```bash
codegraph init -i
```

Regla practica: en trabajo diario no hace falta reindexar a cada cambio pequeño; reindexa tras cambios grandes o al cambiar de contexto de rama/proyecto.

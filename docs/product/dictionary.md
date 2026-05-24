# Diccionario de Producto (Ubiquitous Language)

Este documento define la terminología estándar a utilizar en el diseño, desarrollo, y comunicación del producto, asegurando consistencia entre el equipo técnico y de negocio.

## Términos Centrales (Core)

| Término (UI/Negocio) | Equivalente Técnico | Descripción                                                                                                                                                     |
| :------------------- | :------------------ | :-------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Organización**     | `Tenant`            | Representa a una empresa cliente que adquiere el software. Aísla completamente la facturación, los datos operativos (cartera, contabilidad) y los usuarios.     |
| **Usuario**          | `User`              | Una persona física que accede a la plataforma con un rol específico. Un usuario siempre pertenece a una (o varias) organizaciones.                              |
| _(Evitar)_ Cuenta    | -                   | **NO UTILIZAR** para referirse a la organización del cliente, ya que causa colisión directa con conceptos del dominio (ej. Cuenta Contable, Cuenta por Cobrar). |
| _(Evitar)_ Workspace | -                   | **NO UTILIZAR**. Demasiado informal para el rigor financiero y contable esperado en la plataforma.                                                              |

_Nota Técnica:_ A nivel de infraestructura (AWS), middleware (Go) e interceptores (Angular), el concepto se mantendrá como `Tenant` por ser el estándar de la industria técnica. Sin embargo, cualquier ruta visible al usuario, pantalla, mensaje de error o modelo de dominio expuesto debe utilizar `Organización` o su id/slug.

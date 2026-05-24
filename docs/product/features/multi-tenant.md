# Funcionalidad Multi-Organización (Multi-Tenant)

## Descripción General

Bowerbird está diseñado para servir a múltiples organizaciones (empresas clientes) desde una única plataforma centralizada. Esta funcionalidad permite que diferentes empresas operen sus procesos de cartera y contabilidad de manera completamente independiente, segura y personalizada.

Cada organización en Bowerbird actúa como un entorno cerrado. Los usuarios, los datos financieros, y las operaciones de una organización son invisibles e inaccesibles para el resto de organizaciones que utilizan el software.

## Valor para el Cliente

- **Seguridad y Privacidad:** Garantía absoluta de que los datos financieros (comprobantes, saldos, facturas) no se mezclarán jamás con los de otra empresa gracias al aislamiento físico de bases de datos.
- **Identidad de Marca (White-labeling básico):** Las empresas acceden a la plataforma a través de una URL dedicada y profesional, como `https://mi-empresa.bowerbird.com`.
- **Colaboración Centralizada:** Una vez dentro de la organización, distintos departamentos (contabilidad, cartera, tesorería) pueden colaborar sobre una única fuente de verdad.

## Comportamiento del Sistema

### 1. Acceso y Rutas

- Las organizaciones interactúan con el sistema exclusivamente a través de su **Subdominio Dedicado** (ej. `acme.bowerbird.com`).
- El sistema detecta automáticamente a qué organización intenta acceder el usuario basándose en esta URL.
- No existe el concepto de "elegir empresa" en un menú desplegable global antes de iniciar sesión. El punto de entrada dicta el entorno operativo.

### 2. Aislamiento de Datos

- En el núcleo tecnológico, cada organización posee su propia base de datos exclusiva.
- Un error humano o un bug en el sistema no puede resultar en la filtración de datos cruzados entre organizaciones.
- Las tareas operativas como copias de seguridad (backups), restauraciones, o cumplimiento de normativas de retención de datos pueden realizarse de manera individual por organización.

### 3. Procesamiento Asíncrono

- Las operaciones pesadas o automatizadas que el sistema realiza en segundo plano (ej. cierres contables nocturnos, recordatorios automáticos de cartera) se procesan respetando el aislamiento.
- El sistema es capaz de "recordar" a qué organización pertenece una tarea en segundo plano y ejecutarla en el entorno correcto sin interferir con las cargas de trabajo de otras organizaciones.

## Diccionario de Términos

Para evitar confusiones con la terminología financiera, nos adherimos al siguiente estándar:

- ✅ **Organización:** Se refiere a la entidad, empresa o cliente que usa la plataforma (ej. "La organización 'Acme' ha sido registrada").
- ❌ **Cuenta:** NO utilizar para referirse a empresas, ya que en nuestro dominio se refiere a cuentas contables, cuentas por cobrar o cuentas bancarias.
- ❌ **Espacio de Trabajo / Workspace:** NO utilizar. Demasiado informal.

_(Nota Técnica: A nivel interno de infraestructura e ingeniería, esta funcionalidad se gestiona bajo el patrón estandarizado "Tenant" o "Multi-Tenancy")._

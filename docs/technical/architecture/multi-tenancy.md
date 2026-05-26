# Arquitectura Multi-Tenant

Este documento describe la estrategia de arquitectura y la implementación técnica del soporte multi-tenant en Bowerbird, diseñado para soportar múltiples organizaciones (empresas) operando de manera aislada sobre la misma infraestructura base.

## 1. Patrón Arquitectónico: Database-per-Tenant

Bowerbird implementa el patrón **Database-per-Tenant** (Base de datos por Organización) complementado con una base de datos central de catálogo, siguiendo el modelo de **Control Plane y Data Plane**.

Este enfoque se eligió porque garantiza el aislamiento absoluto de los datos financieros y contables de cada organización, simplifica el cumplimiento de normativas de privacidad, y permite restaurar respaldos de manera individualizada sin afectar a otros clientes.

### 1.1. Control Plane (Plano de Control)

El Control Plane es la base de datos centralizada compartida de la plataforma. Sus responsabilidades son estrictamente a nivel de sistema, no de negocio.

- **Catálogo de Organizaciones (`tenants` table):** Mantiene el registro maestro de cada organización, asociando el subdominio (`slug`) con el nombre de la base de datos física (`db_name`).
- **Resolución:** Cuando entra una petición, el sistema consulta esta base de datos para saber hacia dónde enrutar la conexión.
- **Gestión Global:** Puede almacenar metadatos de facturación del producto, usuarios administradores globales (superadmins de la plataforma) y configuraciones generales.

### 1.2. Data Plane (Plano de Datos)

El Data Plane comprende las bases de datos individuales de cada organización.

- **Aislamiento Físico:** Cada organización tiene su propia base de datos PostgreSQL (ej. `db_tenant_acme`, `db_tenant_stark`).
- **Esquema de Dominio:** Aquí residen todas las tablas operativas de negocio: cuentas contables, facturas, comprobantes, cartera, etc.
- **Seguridad:** Un usuario de una organización solo tiene acceso a las credenciales/conexión de su respectiva base de datos.

---

## 2. Resolución del Tenant (Identidad Dinámica)

La arquitectura híbrida (Subdominio en Frontend + Header en Backend) permite una experiencia personalizada para el usuario manteniendo la API escalable bajo un dominio estático.

### 2.1. Experiencia de Usuario (Frontend - Angular)

- **Rutas por Tenant:** La PWA de Angular estructura la navegación colocando el identificador del tenant en el primer segmento de la ruta (ej. `https://app.bowerbird.com/acme/inbox`).
- **Layout Centralizado:** Todas las rutas asociadas a un tenant (`/:tenantId/*`) están anidadas bajo el `TenantLayoutComponent`, garantizando un contexto visual coherente.
- **Intercepción HTTP:** En la PWA, el archivo `tenant.interceptor.ts` examina la URL activa (`location.pathname`), extrae el primer segmento (si no es una ruta global como `/login`), y lo inyecta en el header HTTP `X-Tenant-ID` en absolutamente todas las llamadas que se realizan hacia la API central (`https://api.bowerbird.com`).

### 2.2. Resolución en API (Backend - Go)

1. **Middleware HTTP:** El `tenant.Middleware` en Go intercepta el header `X-Tenant-ID` de la petición HTTP.
2. **Contexto de Go:** Inyecta este valor en el `context.Context` de la petición. Esto asegura que la identidad del tenant viaje de manera segura a través de los adaptadores, casos de uso y repositorios sin alterar las firmas de las funciones.
3. **Registry Dinámico (`pgxpool`):** Cuando la capa de infraestructura necesita consultar datos, solicita una conexión al `Registry` de base de datos pasando el contexto.
4. **Enrutamiento y Caché:** El `Registry` lee el tenant del contexto, busca en un mapa concurrente (`sync.Map`) si ya existe un Pool de conexiones abierto para ese tenant. Si no existe, consulta el Control Plane, resuelve el `db_name`, abre un nuevo pool, lo guarda en caché y lo devuelve.

---

## 3. Infraestructura y Aislamiento Lógico (AWS CDK)

Mientras que los datos relacionales están aislados físicamente (Database-per-tenant), los recursos de infraestructura como SQS o S3 son compartidos para optimizar costos, utilizando "Aislamiento Lógico".

- **AWS CloudFront & Route53:** Configurado para soportar `*.bowerbird.com` y rutearlo estáticamente al bucket de la PWA, independientemente del subdominio.
- **Colas SQS & EventBridge:** Son recursos compartidos. El _Poller_ de eventos en Go extrae automáticamente el `TenantID` desde los `MessageAttributes` del mensaje de AWS, y lo re-inyecta en el `context.Context` de ejecución. De esta forma, el procesamiento asíncrono tiene las mismas garantías de enrutamiento a la base de datos que una petición HTTP síncrona.
- **CORS Dinámico:** El API Gateway está configurado para aceptar orígenes de `https://*.bowerbird.com`.

---

## 4. Diccionario Ubicuo (Ubiquitous Language)

Para mantener la consistencia entre los requerimientos funcionales y la implementación técnica, aplicamos las siguientes convenciones:

- **En UI, Producto y Negocio:** Se utiliza el término **Organización** (o Empresa). Representa a la entidad corporativa que adquiere el software. Se evitan explícitamente términos como _Cuenta_ (por colisión con conceptos contables) o _Workspace_.
- **En Código, Infraestructura y Headers:** Se utiliza el estándar de la industria **Tenant** (ej. `X-Tenant-ID`, `tenant.Middleware`, tabla `tenants`).

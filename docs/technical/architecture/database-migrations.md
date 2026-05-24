# Gestión de Migraciones (Control Plane y Data Plane)

Bowerbird utiliza `github.com/golang-migrate/migrate/v4` para administrar la evolución del esquema de la base de datos de manera programática y automatizada, respetando la arquitectura Multi-Tenant.

## 1. Estructura de Directorios

Las migraciones SQL están divididas estrictamente en dos carpetas dentro de `apps/backend/migrations`:

- **`controlplane/`**: Contiene las migraciones de la base de datos central compartida (catálogo de organizaciones, configuraciones globales).
- **`tenant/`**: Contiene el esquema de negocio (usuarios, cartera, contabilidad). Estas migraciones se aplican **a cada base de datos aislada** de las organizaciones.

_Ejemplo:_

```
apps/backend/migrations/
├── controlplane/
│   ├── 000001_create_tenants_table.up.sql
│   └── 000001_create_tenants_table.down.sql
└── tenant/
    ├── 000001_init_tenant_schema.up.sql
    └── 000001_init_tenant_schema.down.sql
```

## 2. Herramienta CLI de Migración (`cmd/migrate`)

Se ha desarrollado una herramienta en Go (`apps/backend/cmd/migrate/main.go`) que orquesta el proceso.

### Comandos disponibles (vía `pnpm`):

- **Migrar solo el Control Plane:**

  ```bash
  pnpm run migrate:controlplane
  ```

  _Qué hace:_ Lee el `DATABASE_URL` (vía SSM o `.env`) y ejecuta `golang-migrate` sobre esa base de datos utilizando los archivos de la carpeta `controlplane`.

- **Migrar todos los Tenants activos:**

  ```bash
  pnpm run migrate:tenants
  ```

  _Qué hace:_
  1. Se conecta al Control Plane.
  2. Ejecuta `SELECT db_name FROM tenants WHERE status = 'active'`.
  3. Itera sobre cada base de datos de la lista (ej. `tenant_acme`, `tenant_stark`).
  4. Ejecuta las migraciones de la carpeta `tenant/` individualmente sobre cada una.

- **Migrar todo el sistema (Recomendado para despliegues):**
  ```bash
  pnpm run migrate:all
  ```
  _Qué hace:_ Ejecuta la migración del Control Plane y, si es exitosa, procede a iterar y migrar todos los Tenants.

## 3. Consideraciones para el Ciclo de Vida (Onboarding)

Cuando el sistema registre una nueva Organización (Tenant) en el futuro, el flujo backend deberá:

1. Crear el registro en el Control Plane (tabla `tenants`).
2. Ejecutar físicamente `CREATE DATABASE {db_name}` en PostgreSQL.
3. Importar y llamar a la función `database.RunMigrations(tenantURL, "migrations/tenant")` de manera programática en Go para que esa base de datos recién nacida adopte el esquema más actual de negocio inmediatamente.

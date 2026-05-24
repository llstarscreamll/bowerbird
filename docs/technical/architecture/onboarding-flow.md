# Flujo de Onboarding de Organizaciones

Este documento describe el flujo automatizado de provisión técnica que ocurre cuando una nueva Organización (Tenant) se registra en la plataforma.

## 1. El Reto del Onboarding Multi-Tenant

En una arquitectura de base de datos compartida, registrar un cliente es un simple `INSERT`. En nuestra arquitectura de _Database-per-Tenant_, registrar un cliente implica:

1. Asegurar la disponibilidad de la identidad (Subdominio/Slug).
2. Crear registros en la base de datos central (Control Plane).
3. Aprovisionar infraestructura real al vuelo (Crear la base de datos en Postgres).
4. Sincronizar el esquema de esa nueva base de datos con la versión más reciente del software.

## 2. Implementación del Caso de Uso (`CreateOrganizationUseCase`)

El flujo está centralizado en la capa de aplicación: `apps/backend/internal/organization/application/create.go`.

### Pasos del Flujo:

1.  **Validación de Dominio:** Se crea la entidad `Organization`, validando que el `slug` (subdominio) solo contenga caracteres alfanuméricos y guiones. Se autogenera un nombre seguro para la base de datos (ej. `tenant_acme_corp`).
2.  **Unicidad:** Se consulta al `Repository` (Control Plane) para asegurar que el slug no esté en uso.
3.  **Reserva de Identidad:** Se guarda la Organización en la tabla `tenants` del Control Plane. Esto reserva el slug inmediatamente.
4.  **Provisión Física:** Se invoca al `Provisioner` de infraestructura que ejecuta directamente el comando SQL `CREATE DATABASE db_name` sobre el clúster de Postgres.
5.  **Migración de Esquema:** Una vez la base de datos física existe, el mismo provisioner invoca la herramienta programática `golang-migrate`, la cual se conecta a esa base de datos recién creada y ejecuta todos los archivos de la carpeta `migrations/tenant`, dejándola lista para operar.

## 3. Manejo de Fallos (Resiliencia)

Dado que la creación de bases de datos (`CREATE DATABASE`) no puede ejecutarse dentro de una transacción SQL regular junto con el `INSERT` del Control Plane, existen escenarios de fallo:

- **Fallo al crear la DB:** Si el `INSERT` en el Control Plane fue exitoso pero el `CREATE DATABASE` falla, el sistema devolverá un error HTTP 500. El registro queda en el Control Plane, pero no existe base de datos.
- **Fallo al migrar:** Si la DB se creó pero la migración falló, la base de datos queda vacía o parcialmente migrada.

**Solución Actual:** Nuestra herramienta CLI de migraciones globales (`pnpm run migrate:all`) actúa como un mecanismo de auto-curación. Al ejecutarse en el pipeline de despliegue, detectará los tenants en el Control Plane y re-intentará o completará las migraciones faltantes en sus bases de datos aisladas.

## 4. Endpoint de Registro

El flujo está expuesto a través de un endpoint público REST:

```http
POST /api/v1/organizations
Content-Type: application/json

{
    "name": "Wayne Enterprises",
    "slug": "wayne"
}
```

**Respuesta Exitosa (201 Created):**

```json
{
  "id": "bf0c4207-2151-41e5-bf39-a23e927f797a",
  "name": "Wayne Enterprises",
  "slug": "wayne",
  "status": "active",
  "created_at": "2026-05-23T19:20:35Z"
}
```

_A partir de este momento, el cliente puede navegar a `https://wayne.bowerbird.com` y el sistema enrutará exitosamente sus operaciones hacia su entorno seguro._

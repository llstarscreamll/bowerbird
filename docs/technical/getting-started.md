# Getting started técnico y entorno local

## Requisitos

- Mise
- Caddy (para resolución de DNS local y proxy inverso)
- AWS CLI autenticado (solo para deploy)

Versiones definidas por el repo en `.mise.toml`:

- Node.js `24`
- Go `1.25`
- pnpm `10.16.1`
- Air `latest`

## Setup inicial

```bash
mise install
pnpm install
```

Si quieres ejecutar comandos con el toolchain del repo sin tocar lo global:

```bash
mise x -- pnpm run dev
```

## Variables de entorno

### API local

1. Copiar `apps/backend/.env.example` a `apps/backend/.env`.
2. Copiar `apps/backend/secrets.example.json` a `apps/backend/secrets.json`.
3. Revisar y ajustar los valores locales.

**¿Cuándo usar `.env` y cuándo `secrets.json`?**

La regla general del proyecto es que el **`.env` controla el contenedor del proceso** y el **`secrets.json` controla el negocio y la infraestructura sensible**.

Usa **`.env`** únicamente para variables de infraestructura de ejecución que cambian cómo arranca la app:

- Puerto del servidor local (`PORT`).
- Credenciales AWS de IAM para el SDK (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`).
- Región de AWS (`AWS_REGION`).
- URL de emulación para el SDK (`AWS_ENDPOINT_URL`).
- El path en SSM donde buscar los secretos (`SSM_PARAMETER_NAME`).
- Flags de ejecución de desarrollo (`ENABLE_LOCAL_EVENT_LOOP`).

Usa **`secrets.json`** para cualquier clave secreta, cadena de conexión o recurso creado _dentro_ de AWS que sea sensible:

- URL de la Base de datos (`database_url`).
- Nombres de buckets (`s3_bucket_name`).
- URLs de colas de SQS y EventBridge (`sqs_queue_url`, `eventbridge_queue_url`).
- API Keys de proveedores terceros (`third_party_api_key`).
- Tokens o Salts de encriptación.

**Nota:** La API carga su configuración principal leyendo el `SSM_PARAMETER_NAME` definido en el `.env`. En tu entorno local, LocalStack lee automáticamente tu archivo `secrets.json` en disco y lo inyecta como un `SecureString` dentro de SSM para emular el flujo productivo.

4. Exportar variables si vas a ejecutar comandos directos de Go:

```bash
export $(grep -v '^#' apps/backend/.env | xargs)
```

Variables clave en `.env` para emulación AWS local:

- `AWS_ENDPOINT_URL=http://localhost:4566`
- `ENABLE_LOCAL_EVENT_LOOP=true`
- `SSM_PARAMETER_NAME=/bowerbird/local/secrets`

**Nota:** La API carga su configuración principal (base de datos, colas) desde el parámetro de SSM en el boot. LocalStack lee automáticamente tu `secrets.json` y lo inyecta como un SecureString en SSM al inicializarse.

### Infraestructura CDK

1. Copiar `packages/infra/.env.example` a `packages/infra/.env`.
2. Variables clave:

- `AWS_ACCOUNT_ID`
- `AWS_REGION=us-east-1`
- `ROOT_DOMAIN=money-path.co`
- `APP_SUBDOMAIN=app`
- `API_SUBDOMAIN=api`

## Configuración de DNS local y HTTPS

Para que el enrutamiento de la PWA funcione correctamente en tu entorno local (ej. `app.bowerbird.dev` y `api.bowerbird.dev`), utilizamos **Caddy** y configuramos el archivo `/etc/hosts` de tu máquina.

1. Añade los dominios de desarrollo a tu archivo hosts:

```bash
sudo nano /etc/hosts
```

Añade las siguientes líneas:

```text
127.0.0.1   api.bowerbird.dev
127.0.0.1   app.bowerbird.dev
```

2. Caddy ya está configurado en el `docker-compose.yml` utilizando el archivo `Caddyfile` en la raíz del proyecto. Este proxy inverso redirige el tráfico HTTPS de manera local:

- `app.bowerbird.dev` -> Angular (`4200`)
- `api.bowerbird.dev` -> Go API (`8080`)

### Confiar en el certificado SSL local

Caddy genera certificados HTTPS usando una Autoridad Certificadora (CA) interna. Para que el navegador no te muestre la alerta de "Conexión no segura" (`ERR_CERT_AUTHORITY_INVALID`), debes indicar a tu sistema que confíe en ella.

**Paso 1: Extraer el certificado del contenedor**
Con el entorno en ejecución (`pnpm run dev`), ejecuta en otra terminal:

```bash
docker cp bowerbird-caddy:/data/caddy/pki/authorities/local/root.crt ./bowerbird-local-ca.crt
```

**Paso 2: Registrar el certificado**

**En macOS:**

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./bowerbird-local-ca.crt
```

**En Fedora Linux:**

```bash
sudo cp ./bowerbird-local-ca.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

_(Opcional) Si utilizas Firefox, importa el archivo `bowerbird-local-ca.crt` manualmente desde Ajustes > Privacidad y Seguridad > Ver certificados > Autoridades > Importar._

Una vez instalado, **reinicia tu navegador**.

## Desarrollo local

Ejecuta todo el stack de desarrollo:

```bash
pnpm run dev
```

Esto levanta:

- Postgres, Redis, LocalStack (S3/SQS/EventBridge/SSM) y **Caddy** en Docker
- API Go con live reload (`air -c .air.toml`) sirviendo la API
- Angular dev server sirviendo la web

Una vez levantado, en lugar de acceder a localhost, debes acceder a través de los dominios configurados en tu DNS local para que el enrutamiento y las cookies funcionen correctamente:

- Web (App / Global): `https://app.bowerbird.dev`
- Web (Tenant): `https://app.bowerbird.dev/acme/dashboard` (ejemplo de enrutamiento por path)
- API: `https://api.bowerbird.dev`

LocalStack inicializa automáticamente recursos con `apps/backend/scripts/init-localstack.sh` al arrancar Docker.

## Comandos principales

- `pnpm run infra:up`
- `pnpm run infra:down`
- `pnpm run build`
- `pnpm run test`
- `pnpm run lint`
- `pnpm run format`
- `pnpm run format:check`
- `pnpm run deploy`

## Tooling adicional

Para exploración estructural del código y análisis de impacto, revisar CodeGraph:

- [Tooling: CodeGraph](./tooling/codegraph.md)
- [Tooling: LocalStack](./tooling/localstack.md)

# Estrategia de Autenticación

## Resumen

Bowerbird utiliza una arquitectura de sesión mixta. Este enfoque equilibra la escalabilidad de los JSON Web Tokens (JWT) con la seguridad de las cookies tradicionales (HTTP-only), mitigando proactivamente vulnerabilidades como XSS (Cross-Site Scripting) y CSRF (Cross-Site Request Forgery).

## Flujo de Tokens

Al autenticarse exitosamente (mediante login local, registro o flujo OAuth), el backend emite dos tokens distintos:

### 1. Access Token (JWT)

- **Duración:** Corta (ej. 15 minutos).
- **Entrega:** Se devuelve en el cuerpo JSON de la respuesta HTTP (`POST /api/v1/auth/login-local`, `POST /api/v1/auth/refresh`).
- **Almacenamiento (Frontend):** Mantenido exclusivamente en memoria (SignalStore de Angular). **Nunca** se almacena en `localStorage` o `sessionStorage`.
- **Uso:** Se inyecta en las peticiones salientes hacia la API mediante la cabecera `Authorization: Bearer <token>`.
- **Seguridad:** Su almacenamiento estricto en memoria lo protege contra ataques XSS orientados al robo de credenciales en almacenamiento persistente.

### 2. Refresh Token

- **Duración:** Larga (ej. 7 días).
- **Entrega:** Establecido por el backend utilizando una cabecera `Set-Cookie`.
- **Almacenamiento (Frontend):** Gestionado automáticamente por el navegador como una cookie `HttpOnly`, `Secure` y `SameSite=Strict`.
- **Uso:** El navegador lo adjunta de forma automática y transparente a la ruta `/api/v1/auth/refresh` cuando la aplicación necesita renovar el Access Token.
- **Seguridad:** El flag `HttpOnly` previene la lectura del token mediante JavaScript (neutralizando XSS), mientras que `SameSite=Strict` bloquea ataques CSRF.

## Flujo OAuth (Google/Microsoft)

El flujo de autenticación mediante proveedores externos garantiza que ningún token sea expuesto en parámetros de URL, protegiéndolos de filtraciones en el historial del navegador o logs de red.

1. **Redirección Segura:** Una vez concluido el intercambio OAuth, el backend establece la cookie `HttpOnly` con el `refresh_token` y redirige al usuario directamente a `/lobby` mediante un HTTP 302, sin añadir información sensible en la URL.
2. **Interceptación del Frontend:** El `authGuard` de Angular detecta la navegación a la ruta protegida (`/lobby`).
3. **Renovación Silenciosa:** Al no existir un `access_token` en la memoria tras la recarga, el guard invoca automáticamente `POST /api/v1/auth/refresh`. El navegador adjunta la cookie segura en la petición.
4. **Acceso Concedido:** El backend responde con un nuevo `access_token` en el JSON. El frontend lo guarda en memoria y permite la carga del `/lobby`.

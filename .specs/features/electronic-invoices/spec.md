# Especificación - Gestión de Facturas Electrónicas

## Estado

- Estado: Aprobada para diseño e implementación
- Fecha: 2026-05-25
- Feature ID: EFI
- Idioma: ES

## Objetivo

Construir una gestión de facturas electrónicas modular, robusta y determinista para tenants multi-cuenta, mediante ingesta pull de correo, extracción híbrida XML/LLM, deduplicación por mensaje y CUFE, y persistencia estructurada para consulta web y métricas.

## Decisiones cerradas

1. Estrategia de ingesta: `pull` compatible con múltiples proveedores y múltiples cuentas por tenant.
2. Proveedor LLM inicial: Google Gemini vía API directa.
3. Extensibilidad LLM: arquitectura por adaptadores para cambiar fácilmente a OpenAI/Anthropic u otros.
4. Credenciales LLM: obtenidas desde SSM.
5. Credenciales de cuentas de correo: almacenadas cifradas en Postgres del tenant.
6. Frontend incluido: conexión de cuentas, bandeja unificada, estado de sincronización/sesión.
7. Enrutamiento transversal: coreografía por eventos (sin crear un contexto orquestador central por ahora).

## Requisitos funcionales

### RF-INGESTA (Ingesta y monitoreo de correo)

- [EFI-RF-001] El sistema debe registrar múltiples cuentas de correo por tenant.
- [EFI-RF-002] El sistema debe permitir múltiples cuentas del mismo proveedor por tenant.
- [EFI-RF-003] El sistema debe sincronizar por `pull` cada N minutos por cuenta activa.
- [EFI-RF-004] El sistema debe capturar y guardar metadatos del correo en la BD del tenant.
- [EFI-RF-005] El sistema debe descargar todos los adjuntos del correo.
- [EFI-RF-006] El sistema debe subir adjuntos a S3 y guardar referencias en BD.
- [EFI-RF-007] El sistema debe publicar evento `InboxMessageReceived` por cada correo sincronizado.

### RF-PIPELINE (Descompresión y clasificación)

- [EFI-RF-008] El pipeline debe detectar tipo de archivo (ZIP/XML/PDF/otros).
- [EFI-RF-009] El pipeline debe descomprimir ZIP y procesar su contenido.
- [EFI-RF-010] El pipeline debe agrupar archivos que pertenezcan al mismo documento comercial (pareja XML + PDF cuando exista).
- [EFI-RF-011] El pipeline debe marcar documentos no clasificables sin bloquear la cola.

### RF-EXTRACCION (Motor híbrido)

- [EFI-RF-012] El sistema debe priorizar parser XML nativo UBL 2.1 DIAN.
- [EFI-RF-013] El parser XML debe extraer al menos emisor, receptor, CUFE/UUID, totales, impuestos (TaxTotal), códigos de pago y líneas.
- [EFI-RF-014] En ausencia de XML válido, el sistema debe usar Gemini sobre PDF con salida estructurada estricta (JSON Schema).
- [EFI-RF-015] XML y LLM deben normalizar al mismo modelo interno de factura.
- [EFI-RF-016] Todo dato no mapeado debe persistirse en `raw_data` JSONB para no perder información.

### RF-VALIDACION (Negocio y deduplicación)

- [EFI-RF-017] Debe evitarse duplicación por mensaje de correo ya sincronizado.
- [EFI-RF-018] Si un correo ya fue procesado para facturación, debe omitirse sin alterar archivos ni tablas financieras.
- [EFI-RF-019] Antes de persistir factura, debe validarse CUFE único.
- [EFI-RF-020] Si CUFE ya existe, debe registrarse en logs y omitirse sin efectos secundarios.

### RF-ALMACENAMIENTO (S3 privado multi-tenant)

- [EFI-RF-021] El bucket compartido debe segmentar por tenant y módulo con prefijos estandarizados.
- [EFI-RF-022] Los objetos deben permanecer privados y sin acceso público.
- [EFI-RF-023] El acceso a archivos debe ser vía URL prefirmada emitida por backend autenticado y autorizado por tenant.
- [EFI-RF-024] Debe existir estrategia de idempotencia para evitar cargas duplicadas físicamente.

### RF-PERSISTENCIA (Modelo de datos)

- [EFI-RF-025] La información de correo, pipeline y extracción debe persistirse en tablas del tenant.
- [EFI-RF-026] Cada entidad externa debe incluir columna `raw_data` JSONB.
- [EFI-RF-027] Factura debe persistir cabecera con CUFE, totales, impuestos, referencias de soporte y estado de extracción.
- [EFI-RF-028] Factura debe persistir líneas de detalle vinculadas a cabecera.

### RF-FRONTEND (Autorización y bandeja unificada)

- [EFI-RF-029] Debe existir flujo UI para conectar nuevas cuentas (OAuth2 por proveedor).
- [EFI-RF-030] Debe existir vista unificada de correos sincronizados de todas las cuentas conectadas del tenant.
- [EFI-RF-031] La vista debe mostrar estado por cuenta (activa, error de token, requiere reconexión, pausada).
- [EFI-RF-032] La vista debe ser responsive y con UX moderna tipo bandeja global unificada.

## Requisitos no funcionales

- [EFI-RNF-001] Arquitectura limpia y modular por bounded contexts (`inbox`, `invoicing`).
- [EFI-RNF-002] Comunicación entre módulos por eventos (coreografía), minimizando acoplamiento.
- [EFI-RNF-003] Reintentos con backoff para APIs externas (proveedores de correo, Gemini, S3 cuando aplique).
- [EFI-RNF-004] Procesamiento idempotente y determinista por claves de negocio (provider_message_id, cufe, hash adjunto).
- [EFI-RNF-005] Logging estructurado y trazable por tenant, cuenta, mensaje y documento.
- [EFI-RNF-006] Observabilidad con métricas de sincronización, clasificación, extracción, errores y deduplicación.
- [EFI-RNF-007] Seguridad de secretos vía SSM y cifrado en reposo de tokens de cuentas externas en BD tenant.
- [EFI-RNF-008] Escalabilidad horizontal de workers sin romper atomicidad por mensaje/documento.

## Casos de uso (alto nivel)

- [EFI-UC-001] Usuario conecta dos cuentas Gmail y una Outlook en un tenant; el sistema sincroniza todas sin mezclar datos entre tenants.
- [EFI-UC-002] Correo con ZIP que contiene XML+PDF: se descomprime, se prioriza XML, se persiste factura y líneas.
- [EFI-UC-003] Correo con solo PDF: se usa Gemini, se normaliza y se persiste.
- [EFI-UC-004] Correo repetido: se detecta duplicado y se omite.
- [EFI-UC-005] Factura con CUFE ya existente: se registra evento de deduplicación y se omite escritura financiera.
- [EFI-UC-006] Token de una cuenta expira: UI muestra estado requiere reconexión.

## Criterios de aceptación

1. Todos los RF y RNF tienen trazabilidad a tareas de implementación.
2. El flujo completo correo -> evento -> pipeline -> extracción -> persistencia funciona en entorno local con LocalStack.
3. Existe cobertura de pruebas unitarias para parser XML DIAN y normalizador LLM.
4. Existe al menos una prueba de integración del flujo idempotente con deduplicación por CUFE.
5. La UI permite conectar cuentas, revisar estados y listar correos en vista unificada responsive.

## Fuera de alcance (esta entrega)

- Integración push/webhooks nativos de proveedores de correo.
- Procesamiento de otros tipos documentales (gastos no factura, notificaciones legales, compras) más allá de dejar la arquitectura preparada por eventos.
- Motor avanzado de reglas ML para clasificación semántica de correo.

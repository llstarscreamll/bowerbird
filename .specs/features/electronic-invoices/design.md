# Diseño - Gestión de Facturas Electrónicas (EFI)

## 1. Alcance del diseño

Este diseño cubre:

- Ingesta pull multi-proveedor y multi-cuenta por tenant.
- Pipeline de adjuntos (descompresión, clasificación, agrupación documental).
- Extracción híbrida XML DIAN + fallback Gemini.
- Persistencia e idempotencia.
- UI de conexiones y bandeja unificada.

## 2. Bounded contexts

### 2.1 `inbox` (contexto de integración de correo)

Responsabilidades:

- Gestión de cuentas conectadas por tenant/proveedor.
- OAuth2 y refresco de tokens.
- Pull periódico de mensajes por cuenta activa.
- Descarga de adjuntos y almacenamiento en S3.
- Publicación de eventos de dominio de correo.

No responsabilidades:

- No decide reglas de facturación.
- No parsea XML DIAN ni ejecuta LLM de facturas.

### 2.2 `invoicing` (contexto financiero de facturas)

Responsabilidades:

- Suscribirse a eventos de correo y filtrar candidatos.
- Ejecutar pipeline de documentos.
- Parsear XML UBL 2.1 DIAN.
- Ejecutar fallback Gemini para PDF.
- Normalizar, deduplicar y persistir factura + líneas.

No responsabilidades:

- No gestiona OAuth de correo.
- No sincroniza bandejas directamente.

## 3. Decisiones de arquitectura

1. **Coreografía por eventos**: `inbox` publica `InboxMessageReceived`; `invoicing` consume de forma desacoplada.
2. **Sin contexto orquestador central (por ahora)**: evita acoplamiento y optimización prematura.
3. **Patrón Adapter/Strategy para LLM**: interfaz estable `InvoiceLLMExtractor` con implementación inicial Gemini.
4. **Idempotencia multinivel**: por mensaje (provider_message_id), archivo (hash), y factura (CUFE).
5. **S3 compartido segmentado por prefijos**: aislamiento lógico por tenant/modulo/etapa.

## 4. Flujo principal end-to-end

1. Worker de `inbox` obtiene cuentas activas de un tenant.
2. Por cuenta: ejecuta pull incremental de correos nuevos/no procesados.
3. Guarda metadatos crudos de correo (`raw_data`) y adjuntos en S3.
4. Publica evento `InboxMessageReceived` con referencias S3.
5. Consumer `invoicing` recibe evento y clasifica si aplica a factura.
6. Si aplica: descomprime, clasifica y agrupa documentos.
7. Si hay XML válido: parsea UBL 2.1 (prioridad 1).
8. Si no hay XML: invoca Gemini en PDF (prioridad 2).
9. Normaliza al modelo interno, valida CUFE e idempotencia.
10. Persiste cabecera, líneas y `raw_data`, emitiendo logs/métricas.

## 5. Componentes de aplicación

## 5.1 En `apps/backend/internal/inbox`

- `application/sync_accounts_usecase.go`
  - Orquesta ciclo de sincronización por cuenta.
- `domain/connected_account.go`
  - Entidad de cuenta conectada y estado.
- `domain/email_message.go`
  - Entidad de mensaje sincronizado.
- `domain/attachment.go`
  - Entidad de adjunto con hash, tipo y S3 key.
- `domain/events.go`
  - Evento `InboxMessageReceived`.
- `infra/provider/gmail_client.go` y `.../outlook_client.go` (iterativo)
  - Adaptadores por proveedor.
- `infra/repository/postgres/*.go`
  - Repositorios tenant para cuentas/mensajes.

## 5.2 En `apps/backend/internal/invoicing`

- `application/process_inbox_event_usecase.go`
  - Punto de entrada por evento.
- `application/classify_documents_usecase.go`
  - Clasificación/agrupación documental.
- `application/extract_invoice_usecase.go`
  - Selecciona XML o LLM fallback.
- `domain/invoice.go`, `invoice_line.go`
  - Agregado de factura.
- `domain/extractors.go`
  - Interfaces `InvoiceXMLExtractor`, `InvoiceLLMExtractor`.
- `infra/xml/dian_ubl21_parser.go`
  - Parser nativo UBL 2.1.
- `infra/llm/gemini_extractor.go`
  - Implementación Gemini.
- `infra/storage/s3_reader.go`
  - Lectura de documentos desde S3.
- `infra/repository/postgres/*.go`
  - Persistencia factura y líneas con `raw_data`.

## 5.3 En `apps/pwa`

- `src/app/features/inbox-connections/*`
  - UI de conexión OAuth por proveedor/cuenta.
- `src/app/features/unified-inbox/*`
  - Bandeja unificada responsive multi-proveedor.
- `src/app/features/unified-inbox/components/account-status-chip/*`
  - Indicadores de estado por cuenta.
- `src/app/features/invoices/*`
  - Vista de facturas extraídas (iterativo).

## 6. Esquema de eventos

### Evento: `InboxMessageReceived`

Campos mínimos:

- `event_id`
- `occurred_at`
- `tenant_id`
- `account_id`
- `provider`
- `provider_message_id`
- `message_internal_id`
- `subject`
- `from`
- `received_at`
- `attachment_refs[]` (S3 key, filename, mime, hash)
- `raw_data_ref`

## 7. Modelo de datos (tenant DB)

Tablas propuestas (nombres iniciales):

- `connected_accounts`
  - `id`, `tenant_id`, `provider`, `email`, `status`, `encrypted_credentials`, `last_sync_at`, `last_error`, `raw_data`, timestamps.
- `email_messages`
  - `id`, `tenant_id`, `account_id`, `provider_message_id`, `thread_id`, `subject`, `sender`, `received_at`, `processing_status`, `raw_data`, timestamps.
- `email_attachments`
  - `id`, `tenant_id`, `message_id`, `filename`, `mime_type`, `size_bytes`, `sha256`, `s3_key`, `raw_data`, timestamps.
- `invoice_headers`
  - `id`, `tenant_id`, `source_message_id`, `cufe`, `invoice_number`, `issuer_name`, `receiver_name`, `currency`, `subtotal`, `tax_total`, `grand_total`, `document_ref_s3_key`, `extraction_source` (`xml`|`llm`), `raw_data`, timestamps.
- `invoice_lines`
  - `id`, `tenant_id`, `invoice_header_id`, `line_number`, `description`, `quantity`, `unit_price`, `line_tax_total`, `line_total`, `raw_data`, timestamps.

Índices/constraints claves:

- Unique (`tenant_id`, `provider`, `provider_message_id`) en `email_messages`.
- Unique (`tenant_id`, `cufe`) en `invoice_headers`.
- Unique (`tenant_id`, `sha256`, `s3_key_scope`) opcional para estrategia anti-duplicado físico.

## 8. Estrategia S3 y privacidad

Prefijo recomendado:

`tenant/{tenant_id}/{module}/{stage}/{yyyy}/{mm}/{dd}/{resource_id}/{filename}`

Ejemplos:

- `tenant/t_123/inbox/raw/2026/05/25/msg_abc/factura.zip`
- `tenant/t_123/invoicing/normalized/2026/05/25/inv_999/factura.pdf`

Reglas:

- Bucket privado con block public access.
- Sin ACLs públicas.
- Descarga solo por URL prefirmada emitida por backend autenticado/autorizado.

## 9. Seguridad y secretos

- Tokens de cuentas externas cifrados en BD tenant (`encrypted_credentials`).
- Clave de cifrado gestionada por entorno seguro (KMS/secret de app).
- Credenciales LLM obtenidas desde SSM a través de configuración backend.
- Rotación y manejo de errores de autenticación con estado de cuenta visible en UI.

## 10. Confiabilidad, idempotencia y reintentos

- Retry con backoff exponencial para APIs externas y operaciones transitorias.
- Dead-letter queue para eventos no procesables tras máximos intentos.
- Procesamiento idempotente por:
  - (`tenant_id`, `provider`, `provider_message_id`)
  - (`tenant_id`, `cufe`)
  - (`tenant_id`, `sha256` archivo)
- Transacciones DB por unidad atómica de factura.

## 11. Observabilidad

- Logging estructurado con `tenant_id`, `account_id`, `message_id`, `cufe`, `event_id`, `attempt`.
- Métricas mínimas:
  - `inbox_sync_messages_total`
  - `inbox_sync_errors_total`
  - `invoicing_documents_classified_total`
  - `invoicing_extraction_xml_total`
  - `invoicing_extraction_llm_total`
  - `invoicing_duplicates_skipped_total`
  - `invoicing_processing_latency_ms`

## 12. Evolución futura

- Agregar nuevos consumidores del evento `InboxMessageReceived` (gastos, legal, compras) sin romper `inbox`.
- Agregar adaptadores LLM adicionales sin cambios en casos de uso.
- Evaluar un contexto de enrutamiento dedicado solo cuando existan reglas transversales complejas y compartidas entre 3+ dominios consumidores.

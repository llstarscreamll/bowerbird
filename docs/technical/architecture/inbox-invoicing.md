# Flujo de Recepción y Extracción de Facturas (Inbox & Invoicing)

Este documento describe funcional y técnicamente los contextos de **Inbox** e **Invoicing**, los cuales automatizan la ingesta de correos electrónicos, la identificación de facturas electrónicas colombianas (DIAN) y la extracción de sus datos hacia una base de datos centralizada.

---

## 1. Documentación Funcional

El objetivo principal de estos dos módulos es eliminar la necesidad de que los usuarios ingresen facturas manualmente, permitiendo que el sistema se conecte directamente a sus buzones de correo, extraiga las facturas electrónicas y las procese de manera automatizada.

### 1.1. Gestión de Inbox (Buzón de Entrada)

El contexto de _Inbox_ es responsable de gestionar las conexiones a proveedores de correo y sincronizar los mensajes:

- **Cuentas Conectadas**: Permite vincular cuentas de correo (ej. Gmail). El estado de estas cuentas (activa, requiere reconexión, pausada) se monitorea constantemente.
- **Sincronización Incremental**: Periódicamente, el sistema sincroniza los nuevos correos electrónicos basándose en la fecha de la última sincronización, optimizando así las llamadas al proveedor.
- **Procesamiento de Adjuntos**: Los archivos adjuntos de los correos sincronizados son descargados de manera segura y almacenados temporalmente en el almacenamiento en la nube (S3), listos para ser analizados.

### 1.2. Procesamiento de Facturas (Invoicing)

El contexto de _Invoicing_ actúa sobre los nuevos mensajes recibidos y busca extraer información contable útil:

- **Filtrado de Candidatos**: No todos los correos son facturas. El sistema filtra los correos entrantes mediante reglas simples (palabras clave en el asunto como "factura" o la presencia de adjuntos XML/PDF).
- **Clasificación y Agrupación**: Los adjuntos se analizan para determinar su tipo. Si el adjunto es un archivo ZIP (práctica común de la DIAN), el sistema lo descomprime automáticamente. Luego, agrupa lógicamente los archivos XML y PDF que pertenecen a una misma factura basándose en la similitud de sus nombres.
- **Extracción Inteligente de Datos**:
  - **Vía XML (Estándar DIAN)**: Es la ruta principal y más precisa. El sistema parsea el archivo XML buscando la estructura estándar UBL 2.1 requerida por la DIAN en Colombia, obteniendo totales, impuestos, emisor, receptor y líneas de detalle.
  - **Vía Inteligencia Artificial (PDF)**: Como ruta de contingencia, si no existe un XML o no puede ser leído, el sistema envía el documento PDF a un modelo de IA (Google Gemini) con instrucciones estrictas para extraer la misma estructura de datos de forma predecible y estandarizada.
- **Deduplicación**: Para evitar cobros duplicados o contabilidad errónea, el sistema verifica que la factura no haya sido procesada antes, buscando el mensaje de origen o verificando el **CUFE** (Código Único de Facturación Electrónica).

---

## 2. Documentación Técnica

El sistema está diseñado utilizando los principios de **Domain-Driven Design (DDD)** y **Event-Driven Architecture (EDA)**, dividiendo las responsabilidades en dos Bounded Contexts independientes comunicados por eventos de dominio.

### 2.1. Arquitectura de Eventos

El flujo de datos se conecta mediante el evento `InboxMessageReceived`.

1. `inbox` finaliza la descarga y persistencia temporal del correo y sus adjuntos.
2. `inbox` publica el evento `InboxMessageReceived` indicando el proveedor, ID del mensaje, y referencias (S3 keys) a los archivos adjuntos.
3. `invoicing` suscribe este evento y desencadena su orquestación interna para procesar esa posible factura.

### 2.2. Bounded Context: `inbox`

Ubicado en `apps/backend/internal/inbox`. Se encarga de la integración directa con proveedores de correo.

- **Dominio (`domain/models.go`)**:
  - `ConnectedAccount`: Entidad que representa la vinculación de un usuario con un proveedor. Incluye la gestión de estado de sincronización (Transitions to `active`, `error`, etc.).
  - `EmailMessage` y `EmailAttachment`: Entidades para representar y persistir la metadata del correo y sus anexos descargados.
- **Seguridad (`application/credentials_service.go`)**: Cifra los tokens OAuth de los usuarios en la capa de aplicación antes de persistirlos en la base de datos.
- **Caso de Uso Principal (`SyncAccountsUseCase`)**:
  1.  Recupera cuentas activas.
  2.  Instancia el cliente del proveedor (ej. Gmail) utilizando credenciales descifradas.
  3.  Pide la lista de mensajes incrementales (query `after:UNIX_TIMESTAMP`).
  4.  Descarga los mensajes, almacena los adjuntos en S3 (`AttachmentStorage`) generando el hash SHA256.
  5.  Guarda en base de datos de Inbox y dispara el evento `InboxMessageReceived`.

### 2.3. Bounded Context: `invoicing`

Ubicado en `apps/backend/internal/invoicing`. Transforma "archivos crudos" en un esquema de factura estructurado.

- **Dominio (`domain/invoice.go`, `domain/document_classification.go`)**:
  - `InvoiceDocument`: Agregado principal que mantiene las validaciones de negocio (`Validate()` asegura que exista CUFE, Emisor, Receptor y Líneas).
  - `DocumentClassifier`: Lógica de dominio para detectar si un archivo es ZIP, XML o PDF basándose en _magic bytes_ o extensiones. Descomprime archivos ZIP en memoria y agrupa representaciones (`DocumentGroup`) mediante la normalización de los nombres de archivo.
- **Flujo de Ejecución (Casos de uso)**:
  1.  `ProcessInboxEventUseCase`: Recibe el evento de Inbox. Revisa mediante expresiones regulares ligeras si el asunto o extensiones de archivos sugieren una factura. Si lo es, enruta al procesador.
  2.  `ClassifyDocumentsUseCase`: Descarga de S3 los binarios, los pasa al `InvoiceDocumentClassifier` y recibe grupos lógicos (`DocumentGroup`) listos para extracción.
  3.  `ExtractInvoiceUseCase`: Para cada grupo de documentos:
      - Revisa la base de datos de deduplicación vía `InvoiceDedupRepository` (por mensaje de origen).
      - Decide el origen: Si hay XML, delega a `xmlExtractor`. Si no hay XML pero hay PDF, delega a `llmExtractor`.
      - Vuelve a comprobar la duplicidad, esta vez utilizando el **CUFE** obtenido.
  4.  `PersistInvoiceUseCase`: Traduce el agregado `InvoiceDocument` a registros tabulares (`InvoiceHeaderRecord` y `InvoiceLineRecord`) y los persiste atómicamente.
- **Infraestructura de Extracción**:
  - **DIANUBL21Parser (`infrastructure/xml/dian_ubl21_parser.go`)**: Parsea facturas bajo el estándar técnico DIAN (Colombia). Utiliza el paquete nativo `encoding/xml` de Go con `structs` profundamente anidados correspondientes a las especificaciones UBL 2.1 de la DIAN.
  - **GeminiExtractor (`infrastructure/llm/gemini_extractor.go`)**: Usa Google Gemini (`gemini-2.0-flash`). Envía el PDF codificado en Base64 junto a un prompt restrictivo (System Prompt) y un esquema estricto (JSON Schema en la configuración de generación) para forzar al modelo a devolver un objeto de respuesta parseable por la aplicación sin alucinaciones de formato.

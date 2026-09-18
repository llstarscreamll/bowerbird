import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class ErrorTranslationService {
  private readonly translations: Record<string, string> = {
    ERR_INTERNAL: 'Ha ocurrido un error inesperado en el servidor. Por favor, intenta de nuevo más tarde.',
    ERR_NOT_FOUND: 'No pudimos encontrar el recurso solicitado.',
    ERR_VALIDATION: 'Los datos proporcionados no son válidos. Por favor verifica la información e intenta de nuevo.',
    ERR_UNAUTHORIZED: 'No estás autorizado para realizar esta acción. Verifica que hayas iniciado sesión.',
    ERR_FORBIDDEN: 'No tienes los permisos necesarios para acceder a este recurso.',
    ERR_CONFLICT: 'Hubo un conflicto con el estado actual (ej: el registro ya existe).',
    ERR_NOT_IMPLEMENTED: 'Esta funcionalidad aún no está implementada.',
    ERR_SYNC_REAUTH_REQUIRED: 'La cuenta de correo requiere reconexión para continuar sincronizando.',
    ERR_SYNC_RATE_LIMITED: 'El proveedor de correo limitó temporalmente las solicitudes de sincronización.',
    ERR_SYNC_PROVIDER_TEMPORARY: 'El proveedor de correo no está disponible temporalmente.',
    ERR_SYNC_PAYLOAD_REJECTED: 'Se detectó un correo con contenido no procesable de forma segura y fue omitido.',
    ERR_SYNC_INTERNAL: 'Ocurrió un problema interno al sincronizar la cuenta de correo.',
    // Specific domain errors can be added here
    ERR_NETWORK: 'No se pudo conectar con el servidor. Verifica tu conexión a internet.',
  };

  translate(code?: string, defaultMessage?: string): string {
    if (!code) {
      return defaultMessage || 'Ha ocurrido un error desconocido.';
    }

    return this.translations[code] || defaultMessage || `Error inesperado (${code}).`;
  }
}

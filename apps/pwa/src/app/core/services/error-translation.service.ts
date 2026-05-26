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

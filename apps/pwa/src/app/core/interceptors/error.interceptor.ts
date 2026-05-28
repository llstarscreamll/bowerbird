import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { EnrichedHttpError, JSONAPIError } from '../http/jsonapi-error';
import { ToastService } from '../services/toast.service';
import { ErrorTranslationService } from '../services/error-translation.service';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const toastService = inject(ToastService);
  const translationService = inject(ErrorTranslationService);

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      // 1. Check if it's a network error
      if (error.status === 0) {
        toastService.showError(translationService.translate('ERR_NETWORK'), 'Error de Red');
        return throwError(() => error);
      }

      // 2. Parse JSON:API Error Document if present
      let jsonApiErrors: JSONAPIError[] = [];
      if (error.error && Array.isArray(error.error.errors)) {
        jsonApiErrors = error.error.errors;
      }

      // 3. Process each error
      if (jsonApiErrors.length > 0) {
        jsonApiErrors.forEach((err) => {
          // Log _debug payload for developers/AI agents
          if (err.meta?._debug) {
            console.groupCollapsed(`%c[API Error] ${err.code || 'UNKNOWN'}`, 'color: red; font-weight: bold;');
            console.log('Status:', err.status);
            console.log('Title:', err.title);
            console.log('Detail:', err.detail);
            console.log('Trace ID:', err.id);
            console.log('Debug Meta:', err.meta._debug);
            console.groupEnd();
          }

          const friendlyMessage = translationService.translate(err.code, err.detail);

          // Global Toast for Server Errors (5xx)
          if (error.status >= 500) {
            toastService.showError(friendlyMessage, err.title || 'Error del Servidor');
          }

          // Let the component handle 4xx errors via AlertComponent,
          // but throw an enriched error object for easier catching.
          err.detail = friendlyMessage; // Swap the detail with the translated message so components can just show it
        });

        const enrichedError: EnrichedHttpError = { original: error, jsonApiErrors };
        return throwError(() => enrichedError);
      }

      // Fallback if not JSON:API
      if (error.status >= 500) {
        toastService.showError('Ha ocurrido un error inesperado. Por favor, intenta de nuevo.', `Error ${error.status}`);
      }

      return throwError(() => error);
    }),
  );
};

import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthStore } from '../../auth/application/auth.store';
import { AuthHttpService } from '../../auth/infrastructure/auth.http.service';
import { catchError, switchMap, throwError } from 'rxjs';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const store = inject(AuthStore);
  const authService = inject(AuthHttpService);
  const token = store.accessToken();

  let clonedReq = req;

  if (token) {
    clonedReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  // Ensure withCredentials is true to send cookies for refresh endpoint automatically
  // but generally it's better to configure it specifically or globally.
  if (req.url.includes('/refresh') || req.url.includes('/logout')) {
    clonedReq = clonedReq.clone({
      withCredentials: true,
    });
  }

  return next(clonedReq).pipe(
    catchError((error) => {
      // 401 Unauthorized - Try to refresh
      if (error.status === 401 && !req.url.includes('/refresh')) {
        return authService.refreshToken().pipe(
          switchMap((res) => {
            store.setToken(res.access_token);
            // Retry the original request with the new token
            const retryReq = req.clone({
              setHeaders: {
                Authorization: `Bearer ${res.access_token}`,
              },
            });
            return next(retryReq);
          }),
          catchError((refreshErr) => {
            // Refresh failed, logout
            store.clearToken();
            return throwError(() => refreshErr);
          }),
        );
      }
      return throwError(() => error);
    }),
  );
};

import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthStore } from '../../auth/application/auth.store';
import { catchError, switchMap, throwError } from 'rxjs';
import { requiresCookieAuth } from './http-rules';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const store = inject(AuthStore);
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
  if (requiresCookieAuth(req.url)) {
    clonedReq = clonedReq.clone({
      withCredentials: true,
    });
  }

  return next(clonedReq).pipe(
    catchError((error) => {
      // 401 Unauthorized - Try to refresh
      if (error.status === 401 && !req.url.includes('/refresh')) {
        return store.refreshSession().pipe(
          switchMap((refreshedToken) => {
            if (!refreshedToken) {
              store.clearToken();
              return throwError(() => error);
            }

            // Retry the original request with the new token
            const retryReq = req.clone({
              setHeaders: {
                Authorization: `Bearer ${refreshedToken}`,
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

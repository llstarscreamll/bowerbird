import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthStore } from '../../auth/application/auth.store';
import { catchError, switchMap, throwError } from 'rxjs';
import { requiresCookieAuth, shouldSkipAuthRefresh } from './http-rules';

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

  if (requiresCookieAuth(req.url)) {
    clonedReq = clonedReq.clone({
      withCredentials: true,
    });
  }

  return next(clonedReq).pipe(
    catchError((error) => {
      if (error.status === 401 && !shouldSkipAuthRefresh(req.url)) {
        return store.refreshSession().pipe(
          switchMap((refreshedToken) => {
            if (!refreshedToken) {
              store.clearToken();
              return throwError(() => error);
            }

            const retryReq = req.clone({
              setHeaders: {
                Authorization: `Bearer ${refreshedToken}`,
              },
              withCredentials: requiresCookieAuth(req.url),
            });
            return next(retryReq);
          }),
          catchError((refreshErr) => {
            store.clearToken();
            return throwError(() => refreshErr);
          }),
        );
      }
      return throwError(() => error);
    }),
  );
};

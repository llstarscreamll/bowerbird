import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthStore } from '../../auth/application/auth.store';
import { AuthHttpService } from '../../auth/infrastructure/auth.http.service';
import { catchError, map, of } from 'rxjs';

export const publicGuard: CanActivateFn = (route, state) => {
  const store = inject(AuthStore);
  const router = inject(Router);
  const authService = inject(AuthHttpService);

  if (store.isAuthenticated()) {
    return router.createUrlTree(['/lobby']);
  }

  return authService.refreshToken().pipe(
    map((res) => {
      store.setToken(res.access_token);
      return router.createUrlTree(['/lobby']);
    }),
    catchError(() => {
      return of(true);
    }),
  );
};

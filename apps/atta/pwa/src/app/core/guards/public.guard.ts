import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthStore } from '../../auth/application/auth.store';
import { catchError, map, of } from 'rxjs';

export const publicGuard: CanActivateFn = (route, state) => {
  const store = inject(AuthStore);
  const router = inject(Router);

  if (store.isAuthenticated()) {
    return router.createUrlTree(['/lobby']);
  }

  return store.refreshSession().pipe(
    map((token) => (token ? router.createUrlTree(['/lobby']) : true)),
    catchError(() => of(true)),
  );
};

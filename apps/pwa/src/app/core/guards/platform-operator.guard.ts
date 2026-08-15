import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';
import { AuthStore } from '../../auth/application/auth.store';

export const platformOperatorGuard: CanActivateFn = () => {
  const store = inject(AuthStore);
  const router = inject(Router);

  if (store.currentUser()?.platform_operator) {
    return true;
  }

  return store.fetchMe().pipe(map((user) => (user?.platform_operator ? true : router.createUrlTree(['/lobby']))));
};

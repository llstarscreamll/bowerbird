import { inject } from '@angular/core';
import { patchState, signalStore, withMethods, withState } from '@ngrx/signals';
import { rxMethod } from '@ngrx/signals/rxjs-interop';
import { Observable, catchError, map, of, pipe, switchMap, tap } from 'rxjs';
import { TenantMembership } from '../domain/auth.model';
import { AUTH_REPOSITORY } from '../domain/auth.repository';

interface AuthState {
  accessToken: string | null;
  isAuthenticated: boolean;
  tenants: TenantMembership[];
  isLoading: boolean;
  error: string | null;
}

const initialState: AuthState = {
  accessToken: null,
  isAuthenticated: false,
  tenants: [],
  isLoading: false,
  error: null,
};

export const AuthStore = signalStore(
  { providedIn: 'root' },
  withState(initialState),
  withMethods((store, authRepo = inject(AUTH_REPOSITORY)) => ({
    setToken(token: string) {
      patchState(store, { accessToken: token, isAuthenticated: true, error: null });
    },
    clearToken() {
      patchState(store, { accessToken: null, isAuthenticated: false, tenants: [] });
    },
    refreshSession(): Observable<string | null> {
      return authRepo.refreshToken().pipe(
        tap((res) => {
          patchState(store, {
            accessToken: res.access_token,
            isAuthenticated: true,
            error: null,
          });
        }),
        map((res) => res.access_token),
        catchError(() => {
          patchState(store, { accessToken: null, isAuthenticated: false, tenants: [] });
          return of(null);
        }),
      );
    },
    loginLocal: rxMethod<{ email: string; password: string; onSuccess?: () => void }>(
      pipe(
        tap(() => patchState(store, { isLoading: true, error: null })),
        switchMap((credentials) =>
          authRepo.loginLocal(credentials.email, credentials.password).pipe(
            tap((response) => {
              patchState(store, {
                accessToken: response.access_token,
                isAuthenticated: true,
                isLoading: false,
              });
              credentials.onSuccess?.();
            }),
            catchError(() => {
              patchState(store, { error: 'Login failed', isLoading: false });
              return of(null);
            }),
          ),
        ),
      ),
    ),
    loadTenants: rxMethod<void>(
      pipe(
        tap(() => patchState(store, { isLoading: true })),
        switchMap(() =>
          authRepo.getUserTenants().pipe(
            tap((tenants) => {
              patchState(store, { tenants, isLoading: false });
            }),
            catchError(() => {
              patchState(store, { error: 'Failed to load tenants', isLoading: false });
              return of(null);
            }),
          ),
        ),
      ),
    ),
    logout: rxMethod<{ onFinish?: () => void }>(
      pipe(
        tap(() => patchState(store, { isLoading: true })),
        switchMap((options) =>
          authRepo.logout().pipe(
            tap(() => {
              patchState(store, {
                accessToken: null,
                isAuthenticated: false,
                tenants: [],
                isLoading: false,
              });
              options.onFinish?.();
            }),
            catchError(() => {
              patchState(store, { accessToken: null, isAuthenticated: false, isLoading: false });
              options.onFinish?.();
              return of(null);
            }),
          ),
        ),
      ),
    ),
  })),
);

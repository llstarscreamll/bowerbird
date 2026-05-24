import { inject } from '@angular/core';
import { patchState, signalStore, withMethods, withState } from '@ngrx/signals';
import { rxMethod } from '@ngrx/signals/rxjs-interop';
import { pipe, tap, switchMap, catchError, of } from 'rxjs';
import { AuthHttpService } from '../infrastructure/auth.http.service';
import { TenantMembership } from '../domain/auth.model';

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
  withMethods((store, authHttp = inject(AuthHttpService)) => ({
    setToken(token: string) {
      patchState(store, { accessToken: token, isAuthenticated: true, error: null });
    },
    clearToken() {
      patchState(store, { accessToken: null, isAuthenticated: false, tenants: [] });
    },
    loginLocal: rxMethod<{ email: string; password: string }>(
      pipe(
        tap(() => patchState(store, { isLoading: true, error: null })),
        switchMap((credentials) =>
          authHttp.loginLocal(credentials.email, credentials.password).pipe(
            tap((response) => {
              patchState(store, {
                accessToken: response.access_token,
                isAuthenticated: true,
                isLoading: false,
              });
            }),
            catchError((err) => {
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
          authHttp.getUserTenants().pipe(
            tap((tenants) => {
              patchState(store, { tenants, isLoading: false });
            }),
            catchError((err) => {
              patchState(store, { error: 'Failed to load tenants', isLoading: false });
              return of(null);
            }),
          ),
        ),
      ),
    ),
    logout: rxMethod<void>(
      pipe(
        tap(() => patchState(store, { isLoading: true })),
        switchMap(() =>
          authHttp.logout().pipe(
            tap(() => {
              patchState(store, {
                accessToken: null,
                isAuthenticated: false,
                tenants: [],
                isLoading: false,
              });
            }),
            catchError(() => {
              patchState(store, { accessToken: null, isAuthenticated: false, isLoading: false });
              return of(null);
            }),
          ),
        ),
      ),
    ),
  })),
);

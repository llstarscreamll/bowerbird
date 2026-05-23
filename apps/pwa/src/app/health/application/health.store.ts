import { computed, inject } from '@angular/core';
import { patchState, signalStore, withComputed, withMethods, withState } from '@ngrx/signals';
import { rxMethod } from '@ngrx/signals/rxjs-interop';
import { pipe, switchMap, tap } from 'rxjs';
import { HealthStatus } from '../domain/health.model';
import { HEALTH_REPOSITORY } from '../domain/health.repository';

interface HealthState {
  status: HealthStatus;
  isLoading: boolean;
  lastChecked: Date | null;
}

const initialState: HealthState = {
  status: 'checking...',
  isLoading: false,
  lastChecked: null,
};

export const HealthStore = signalStore(
  { providedIn: 'root' },
  withState(initialState),

  withComputed(({ status }) => ({
    isHealthy: computed(() => status() === 'ok'),
  })),

  withMethods((store, healthRepo = inject(HEALTH_REPOSITORY)) => ({
    checkHealth: rxMethod<void>(
      pipe(
        tap(() => patchState(store, { isLoading: true })),
        switchMap(() =>
          healthRepo.checkHealth().pipe(
            tap((healthInfo) => {
              patchState(store, {
                status: healthInfo.status,
                isLoading: false,
                lastChecked: healthInfo.lastChecked,
              });
            }),
          ),
        ),
      ),
    ),
  })),
);

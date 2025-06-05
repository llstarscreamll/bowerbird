import { provideEffects } from '@ngrx/effects';
import { provideStore } from '@ngrx/store';

import { HTTP_INTERCEPTORS, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { ApplicationConfig, LOCALE_ID, provideZoneChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';

import { routes } from '@app/app.routes';
import { metaReducers, reducers } from '@app/ngrx';
import { Effects as AuthEffects } from '@app/ngrx/auth';
import { Effects as FinanceEffects } from '@app/ngrx/finance';
import { WithCredentialInterceptor } from '@app/with-credential.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideHttpClient(withInterceptorsFromDi()),
    provideEffects(AuthEffects, FinanceEffects),
    provideRouter(routes),
    provideStore(reducers, { metaReducers }),
    provideZoneChangeDetection({ eventCoalescing: true }),
    {
      provide: HTTP_INTERCEPTORS,
      useClass: WithCredentialInterceptor,
      multi: true,
    },
    { provide: LOCALE_ID, useValue: 'es-CO' },
  ],
};

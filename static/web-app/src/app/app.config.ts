import { provideStore } from '@ngrx/store';
import { provideEffects } from '@ngrx/effects';
import { provideRouter } from '@angular/router';
import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';

import { routes } from './app.routes';
import { reducers, metaReducers } from './ngrx';
import { Effects as AuthEffects } from './ngrx/auth';
import {
  HTTP_INTERCEPTORS,
  provideHttpClient,
  withInterceptorsFromDi,
} from '@angular/common/http';
import { WithCredentialInterceptor } from './with-credential.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideHttpClient(withInterceptorsFromDi()),
    provideEffects(AuthEffects),
    provideRouter(routes),
    provideStore(reducers, { metaReducers }),
    provideZoneChangeDetection({ eventCoalescing: true }),
    {
      provide: HTTP_INTERCEPTORS,
      useClass: WithCredentialInterceptor,
      multi: true,
    },
  ],
};

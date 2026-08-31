import { ApplicationConfig, isDevMode, provideZonelessChangeDetection } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter, withComponentInputBinding, withRouterConfig } from '@angular/router';
import { provideServiceWorker } from '@angular/service-worker';
import { routes } from './app.routes';
import { AUTH_REPOSITORY } from './auth/domain/auth.repository';
import { AuthHttpService } from './auth/infrastructure/auth.http.service';
import { tenantInterceptor } from './core/interceptors/tenant.interceptor';
import { authInterceptor } from './core/interceptors/auth.interceptor';
import { errorInterceptor } from './core/interceptors/error.interceptor';
import { CONNECTIONS_REPOSITORY } from './connections/domain/connections.repository';
import { ConnectionsHttpRepository } from './connections/infrastructure/connections.http.repository';
import { UNIFIED_INBOX_REPOSITORY } from './inbox/domain/unified-inbox.repository';
import { UnifiedInboxHttpRepository } from './inbox/infrastructure/unified-inbox.http.repository';
import { appIcons } from './shared/icons/app-icons';
import { provideAnalytics } from './core/analytics';
import { providePwaInstall } from './core/pwa-install';
import { provideSystemNotices } from '@bowerbird/system-notices/angular';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideRouter(routes, withComponentInputBinding(), withRouterConfig({ paramsInheritanceStrategy: 'always' })),
    provideHttpClient(withInterceptors([errorInterceptor, authInterceptor, tenantInterceptor])),
    { provide: AUTH_REPOSITORY, useClass: AuthHttpService },
    { provide: CONNECTIONS_REPOSITORY, useClass: ConnectionsHttpRepository },
    { provide: UNIFIED_INBOX_REPOSITORY, useClass: UnifiedInboxHttpRepository },
    appIcons,
    provideAnalytics(),
    provideSystemNotices(),
    providePwaInstall(),
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      registrationStrategy: 'registerWhenStable:30000',
    }),
  ],
};
